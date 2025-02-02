/*
Copyright 2023 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sparseindex

import (
	"fmt"
	"sync"

	"github.com/openGemini/openGemini/engine/hybridqp"
	"github.com/openGemini/openGemini/engine/immutable/colstore"
	"github.com/openGemini/openGemini/lib/errno"
	"github.com/openGemini/openGemini/lib/fragment"
	"github.com/openGemini/openGemini/lib/index"
	"github.com/openGemini/openGemini/lib/logger"
	"github.com/openGemini/openGemini/lib/record"
	"github.com/openGemini/openGemini/lib/rpn"
	"github.com/openGemini/openGemini/lib/util"
	"github.com/openGemini/openGemini/lib/util/lifted/influx/influxql"
	"github.com/openGemini/openGemini/lib/util/lifted/logparser"
	"github.com/openGemini/openGemini/lib/util/lifted/vm/protoparser/influx"
)

// SKIndexReader as a skip index read interface.
type SKIndexReader interface {
	// CreateSKFileReaders generates SKFileReaders for each index field based on the skip index information and condition
	// which is used to quickly determine whether a fragment meets the condition.
	CreateSKFileReaders(option hybridqp.Options, mstInfo *influxql.Measurement, isCache bool) ([]SKFileReader, error)
	// Scan is used to filter fragment ranges based on the secondary key in the condition.
	Scan(reader SKFileReader, rgs fragment.FragmentRanges) (fragment.FragmentRanges, error)
	// Close is used to close the SKIndexReader
	Close() error
}

type TsspFile interface {
	Name() string
	Path() string
}

// SKFileReader as an executor of skip index data reading that corresponds to the index field in the query.
type SKFileReader interface {
	// MayBeInFragment determines whether a fragment in a file meets the query condition.
	MayBeInFragment(fragId uint32) (bool, error)
	// ReInit is used to that a SKFileReader is reused among multiple files.
	ReInit(file interface{}) error
	// Close is used to close the SKFileReader
	Close() error
}

type SKIndexReaderImpl struct {
	property *IndexProperty
	logger   *logger.Logger
}

func NewSKIndexReader(rowsNumPerFragment int, coarseIndexFragment int, minRowsForSeek int) *SKIndexReaderImpl {
	return &SKIndexReaderImpl{
		property: NewIndexProperty(rowsNumPerFragment, coarseIndexFragment, minRowsForSeek),
		logger:   logger.NewLogger(errno.ModuleIndex),
	}
}

func (r *SKIndexReaderImpl) Scan(
	reader SKFileReader,
	rgs fragment.FragmentRanges,
) (fragment.FragmentRanges, error) {
	var (
		droppedFragment uint32 // Number of filtered fragments by skip index
		totalFragment   uint32 // Number of filtered fragments by primary index
		res             fragment.FragmentRanges
	)
	minMarksForSeek := (r.property.MinRowsForSeek + r.property.RowsNumPerFragment - 1) / r.property.RowsNumPerFragment
	for i := 0; i < len(rgs); i++ {
		mr := rgs[i]
		totalFragment += mr.End - mr.Start
		for j := mr.Start; j < mr.End; j++ {
			ok, err := reader.MayBeInFragment(j)
			if err != nil {
				return nil, err
			}
			if !ok {
				droppedFragment++
				continue
			}
			dataRange := fragment.NewFragmentRange(
				util.MaxUint32(rgs[i].Start, j),
				util.MinUint32(rgs[i].End, j+1),
			)
			if len(res) == 0 || int(res[len(res)-1].End-dataRange.Start) > minMarksForSeek {
				res = append(res, dataRange)
			} else {
				res[len(res)-1].End = dataRange.End
			}
		}
	}
	return res, nil
}

func (r *SKIndexReaderImpl) CreateSKFileReaders(option hybridqp.Options, mstInfo *influxql.Measurement, isCache bool) ([]SKFileReader, error) {
	skIndexRelation := mstInfo.IndexRelation
	if skIndexRelation == nil || len(skIndexRelation.Oids) == 0 || option.GetCondition() == nil {
		return nil, nil
	}

	skFieldMap := make(map[string][]string)
	for i, indexList := range skIndexRelation.IndexList {
		// time cluster takes effect on the primary key, not the skip index.
		if skIndexRelation.Oids[i] == uint32(index.TimeCluster) {
			continue
		}
		for _, field := range indexList.IList {
			if _, ok := skFieldMap[field]; !ok {
				skFieldMap[field] = []string{skIndexRelation.IndexNames[i]}
			} else {
				skFieldMap[field] = append(skFieldMap[field], skIndexRelation.IndexNames[i])
			}
		}
	}
	if len(skFieldMap) == 0 {
		return nil, nil
	}
	rpnExpr := rpn.ConvertToRPNExpr(option.GetCondition())
	skInfoMap, err := r.getSKInfoByExpr(rpnExpr, skIndexRelation, skFieldMap)
	if err != nil {
		return nil, err
	}
	return r.createSKFileReaders(skInfoMap, rpnExpr, option, isCache)
}

func (r *SKIndexReaderImpl) getSKInfoByExpr(rpnExpr *rpn.RPNExpr, skIndexRelation *influxql.IndexRelation, skFieldMap map[string][]string) (map[string]*SkInfo, error) {
	skInfoMap := make(map[string]*SkInfo)
	for _, expr := range rpnExpr.Val {
		v, ok := expr.(*influxql.VarRef)
		if !ok {
			continue
		}
		if v.Val == logparser.DefaultFieldForFullText {
			fields := skIndexRelation.GetFullTextColumns()
			if len(fields) == 0 {
				return nil, fmt.Errorf("empty fields for full text index")
			}
			schemas := make([]record.Field, 0, len(fields))
			for i := 0; i < len(fields); i++ {
				schemas = append(schemas, record.Field{Name: fields[i], Type: influx.Field_Type_String})
			}
			// TODO: indexName is used to uniquely identify an index.
			skInfoMap[index.BloomFilterFullTextIndex] = &SkInfo{fields: schemas, oid: uint32(index.BloomFilterFullText)}
			continue
		}
		indexNames, ok := skFieldMap[v.Val]
		if !ok {
			continue
		}
		for i := range indexNames {
			if info, ok := skInfoMap[indexNames[i]]; ok {
				info.fields = append(info.fields, record.Field{Name: v.Val, Type: record.ToModelTypes(v.Type)})
			} else {
				oid, ok := skIndexRelation.GetIndexOidByName(indexNames[i])
				if !ok {
					return nil, fmt.Errorf("invalid the index name %s", indexNames[i])
				}
				skInfoMap[indexNames[i]] = &SkInfo{fields: []record.Field{{Name: v.Val, Type: record.ToModelTypes(v.Type)}}, oid: oid}
			}
		}
	}
	return skInfoMap, nil
}

func (r *SKIndexReaderImpl) createSKFileReaders(skInfoMap map[string]*SkInfo, rpnExpr *rpn.RPNExpr, option hybridqp.Options, isCache bool) ([]SKFileReader, error) {
	var readers []SKFileReader
	for _, v := range skInfoMap {
		indexName, _ := index.GetIndexNameByType(index.IndexType(v.oid))
		creator, ok := GetSKFileReaderFactoryInstance().Find(indexName)
		if !ok {
			return nil, fmt.Errorf("unsupported the skip index type: %s", indexName)
		}
		reader, err := creator.CreateSKFileReader(rpnExpr, v.fields, option, isCache)
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return readers, nil
}

func (r *SKIndexReaderImpl) Close() error {
	return nil
}

type SkInfo struct {
	fields record.Schemas
	oid    uint32
}

// SKFileReaderCreator is used to abstract SKFileReader implementation of multiple skip indexes in factory mode.
type SKFileReaderCreator interface {
	CreateSKFileReader(rpnExpr *rpn.RPNExpr, schema record.Schemas, option hybridqp.Options, isCache bool) (SKFileReader, error)
}

// RegistrySKFileReaderCreator is used to registry the SKFileReaderCreator
func RegistrySKFileReaderCreator(name string, creator SKFileReaderCreator) bool {
	factory := GetSKFileReaderFactoryInstance()

	_, ok := factory.Find(name)
	if ok {
		return ok
	}

	factory.Add(name, creator)
	return true
}

type SKFileReaderCreatorFactory struct {
	creators map[string]SKFileReaderCreator
}

func NewSKFileReaderCreatorFactory() *SKFileReaderCreatorFactory {
	return &SKFileReaderCreatorFactory{creators: make(map[string]SKFileReaderCreator)}
}

func (s *SKFileReaderCreatorFactory) Add(name string, creator SKFileReaderCreator) {
	s.creators[name] = creator
}

func (s *SKFileReaderCreatorFactory) Find(name string) (SKFileReaderCreator, bool) {
	creator, ok := s.creators[name]
	return creator, ok
}

var SKFileReaderInstance *SKFileReaderCreatorFactory
var SKFileReaderOnce sync.Once

func GetSKFileReaderFactoryInstance() *SKFileReaderCreatorFactory {
	SKFileReaderOnce.Do(func() {
		SKFileReaderInstance = NewSKFileReaderCreatorFactory()
	})
	return SKFileReaderInstance
}

type SkipIndex struct {
	bfIdx, fullTextIdx int     // if bloomFilter/fullTextIdx exist, the idx in schemaIdxes is bfIdx/fullTextIdx
	idxInRelation      []int   // mark the position of each skipIndex in the indexRelation, indexRelation may include text、fieldIndex or mergeSet
	schemaIdxes        [][]int // each skip index contains a schemaIdx to indicate the corresponding position of the index column, but under schemaLess, its content may be empty.
	skipIndexWriters   []SkipIndexWriter
}

func NewSkipIndex() *SkipIndex {
	return &SkipIndex{
		bfIdx:       -1,
		fullTextIdx: -1,
	}
}

func (s *SkipIndex) NewSkipIndexWriters(dir, msName, dataFilePath, lockPath string, indexRelation influxql.IndexRelation) {
	s.skipIndexWriters = make([]SkipIndexWriter, 0, len(indexRelation.Oids))
	s.idxInRelation = make([]int, 0, len(indexRelation.Oids))
	pos := 0
	for i := range indexRelation.Oids {
		if indexRelation.IsSkipIndex(i) {
			if indexRelation.IndexNames[i] == index.BloomFilterIndex {
				s.bfIdx = pos
			}
			if indexRelation.IndexNames[i] == index.BloomFilterFullTextIndex {
				s.fullTextIdx = pos
			}

			s.skipIndexWriters = append(s.skipIndexWriters, NewSkipIndexWriter(dir, msName, dataFilePath, lockPath, indexRelation.IndexNames[i]))
			s.idxInRelation = append(s.idxInRelation, i)
			pos++
		}
	}
}

// GetBfIdx get idx in schemaIdxes if bloom filter index exist bfIdx >= 0
func (s *SkipIndex) GetBfIdx() int {
	return s.bfIdx
}

// GetFullTextIdx get idx in schemaIdxes if full text index exist fullTextIdx >= 0
func (s *SkipIndex) GetFullTextIdx() int {
	return s.fullTextIdx
}

func (s *SkipIndex) GetSchemaIdxes() [][]int {
	return s.schemaIdxes
}

func (s *SkipIndex) GetSkipIndexWriters() []SkipIndexWriter {
	return s.skipIndexWriters
}

func (s *SkipIndex) GenSchemaIdxes(schema record.Schemas, indexRelation influxql.IndexRelation) {
	schemaMap := genSchemaMap(schema)
	s.schemaIdxes = make([][]int, 0, len(s.skipIndexWriters))
	var cols []string
	for i := range s.idxInRelation {
		cols = indexRelation.IndexList[s.idxInRelation[i]].IList
		res := make([]int, 0, len(cols))
		// if full text idx exist then get all string col into schema
		if i == s.fullTextIdx {
			for j := range schema {
				if schema[j].IsString() {
					res = append(res, j)
				}
			}
			s.schemaIdxes = append(s.schemaIdxes, res)
			continue
		}

		for idx := range cols {
			_, ok := schemaMap[cols[idx]]
			if !ok {
				// skip missing col
				continue
			}
			res = append(res, schemaMap[cols[idx]])
		}
		s.schemaIdxes = append(s.schemaIdxes, res)
	}
}

func genSchemaMap(schema record.Schemas) map[string]int {
	schemaMap := make(map[string]int)
	for i := range schema {
		schemaMap[schema[i].Name] = i
	}
	return schemaMap
}

func writeSkipIndexToDisk(data []byte, lockPath, skipIndexFilePath string) error {
	indexBuilder := colstore.NewSkipIndexBuilder(&lockPath, skipIndexFilePath)
	err := indexBuilder.WriteData(data)
	defer indexBuilder.Reset()
	return err
}

type SkipIndexWriter interface {
	Open() error
	Close() error
	CreateAttachSkipIndex(schemaIdx, rowsPerSegment []int, writeRec *record.Record) error
	CreateDetachSkipIndex(writeRec *record.Record, schemaIdx, rowsPerSegment []int, dataBuf [][]byte) ([][]byte, []string)
}

func NewSkipIndexWriter(dir, msName, dataFilePath, lockPath, indexType string) SkipIndexWriter {
	switch indexType {
	case index.BloomFilterIndex:
		return NewBloomFilterWriter(dir, msName, dataFilePath, lockPath)
	case index.BloomFilterFullTextIndex:
		return NewFullTextIdxWriter(dir, msName, dataFilePath, lockPath)
	case index.SetIndex:
		return NewSetWriter(dir, msName, dataFilePath, lockPath)
	case index.MinMaxIndex:
		return NewMinMaxWriter(dir, msName, dataFilePath, lockPath)
	default:
		logger.GetLogger().Error("unknown skip index type")
		return nil
	}
}

type skipIndexWriter struct {
	dir, msName            string
	dataFilePath, lockPath string
}

func newSkipIndexWriter(dir, msName, dataFilePath, lockPath string) *skipIndexWriter {
	return &skipIndexWriter{
		dir:          dir,
		msName:       msName,
		dataFilePath: dataFilePath,
		lockPath:     lockPath,
	}
}
