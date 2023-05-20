/*
 * Licensed to the AcmeStack under one or more contributor license
 * agreements. See the NOTICE file distributed with this work for
 * additional information regarding copyright ownership.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gplus

import (
	"database/sql"
	"reflect"

	"github.com/acmestack/gorm-plus/constants"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

var globalDb *gorm.DB
var defaultBatchSize = 1000

func Init(db *gorm.DB) {
	globalDb = db
}

type Page[T any] struct {
	Current int
	Size    int
	Total   int64
	Records []*T
}

type Dao[T any] struct{}

func (dao Dao[T]) NewQuery() (*Query[T], *T) {
	q := &Query[T]{}
	return q, q.buildColumnNameMap()
}

func NewPage[T any](current, size int) *Page[T] {
	return &Page[T]{Current: current, Size: size}
}

// Insert 插入一条记录
func Insert[T any](entity *T, opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	resultDb := db.Create(entity)
	return resultDb
}

// InsertBatch 批量插入多条记录
func InsertBatch[T any](entities []*T, opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	if len(entities) == 0 {
		return db
	}
	resultDb := db.CreateInBatches(entities, defaultBatchSize)
	return resultDb
}

// InsertBatchSize 批量插入多条记录
func InsertBatchSize[T any](entities []*T, batchSize int, opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	if len(entities) == 0 {
		return db
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	resultDb := db.CreateInBatches(entities, batchSize)
	return resultDb
}

// DeleteById 根据 ID 删除记录
func DeleteById[T any](id any, opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	var entity T
	resultDb := db.Where(getPkColumnName[T](), id).Delete(&entity)
	return resultDb
}

// DeleteByIds 根据 ID 批量删除记录
func DeleteByIds[T any](ids any, opts ...OptionFunc) *gorm.DB {
	q, _ := NewQuery[T]()
	q.In(getPkColumnName[T](), ids)
	resultDb := Delete[T](q, opts...)
	return resultDb
}

// Delete 根据条件删除记录
func Delete[T any](q *Query[T], opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	var entity T
	resultDb := db.Where(q.QueryBuilder.String(), q.QueryArgs...).Delete(&entity)
	return resultDb
}

// DeleteByMap 根据Map删除记录
func DeleteByMap[T any](q *Query[T], opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	for k, v := range q.ConditionMap {
		columnName := getColumnName(k)
		q.Eq(columnName, v)
	}
	var entity T
	resultDb := db.Where(q.QueryBuilder.String(), q.QueryArgs...).Delete(&entity)
	return resultDb
}

// UpdateById 根据 ID 更新
func UpdateById[T any](entity *T, opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	resultDb := db.Model(entity).Updates(entity)
	return resultDb
}

// Update 根据 Map 更新
func Update[T any](q *Query[T], opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	resultDb := db.Model(new(T)).Where(q.QueryBuilder.String(), q.QueryArgs...).Updates(&q.UpdateMap)
	return resultDb
}

// SelectById 根据 ID 查询单条记录
func SelectById[T any](id any, opts ...OptionFunc) (*T, *gorm.DB) {
	q, _ := NewQuery[T]()
	q.Eq(getPkColumnName[T](), id)
	var entity T
	resultDb := buildCondition(q, opts...)
	return &entity, resultDb.First(&entity)
}

// SelectByIds 根据 ID 查询多条记录
func SelectByIds[T any](ids any, opts ...OptionFunc) ([]*T, *gorm.DB) {
	q, _ := NewQuery[T]()
	q.In(getPkColumnName[T](), ids)
	return SelectList[T](q, opts...)
}

// SelectOne 根据条件查询单条记录
func SelectOne[T any](q *Query[T], opts ...OptionFunc) (*T, *gorm.DB) {
	var entity T
	resultDb := buildCondition(q, opts...)
	return &entity, resultDb.First(&entity)
}

// Exists 根据条件判断记录是否存在
func Exists[T any](q *Query[T], opts ...OptionFunc) (bool, error) {
	_, dbRes := SelectOne[T](q, opts...)
	return dbRes.RowsAffected > 0, dbRes.Error
}

// SelectList 根据条件查询多条记录
func SelectList[T any](q *Query[T], opts ...OptionFunc) ([]*T, *gorm.DB) {
	resultDb := buildCondition(q, opts...)
	var results []*T
	resultDb.Find(&results)
	return results, resultDb
}

// SelectListModel 根据条件查询多条记录
// 第一个泛型代表数据库表实体
// 第二个泛型代表返回记录实体
func SelectListModel[T any, R any](q *Query[T], opts ...OptionFunc) ([]*R, *gorm.DB) {
	resultDb := buildCondition(q, opts...)
	var results []*R
	resultDb.Scan(&results)
	return results, resultDb
}

// SelectListByMap 根据 Map 查询多条记录
func SelectListByMap[T any](q *Query[T], opts ...OptionFunc) ([]*T, *gorm.DB) {
	resultDb := buildCondition(q, opts...)
	var results []*T
	resultDb.Find(&results)
	return results, resultDb
}

// SelectListMaps 根据条件查询，返回Map记录
func SelectListMaps[T any](q *Query[T], opts ...OptionFunc) ([]map[string]any, *gorm.DB) {
	resultDb := buildCondition(q, opts...)
	var results []map[string]any
	resultDb.Find(&results)
	return results, resultDb
}

// SelectPage 根据条件分页查询记录
func SelectPage[T any](page *Page[T], q *Query[T], opts ...OptionFunc) (*Page[T], *gorm.DB) {
	total, countDb := SelectCount[T](q, opts...)
	if countDb.Error != nil {
		return page, countDb
	}
	page.Total = total
	resultDb := buildCondition(q, opts...)
	var results []*T
	resultDb.Scopes(paginate(page)).Find(&results)
	page.Records = results
	return page, resultDb
}

// SelectPageModel 根据条件分页查询记录
// 第一个泛型代表数据库表实体
// 第二个泛型代表返回记录实体
func SelectPageModel[T any, R any](page *Page[R], q *Query[T], opts ...OptionFunc) (*Page[R], *gorm.DB) {
	total, countDb := SelectCount[T](q, opts...)
	if countDb.Error != nil {
		return page, countDb
	}
	page.Total = total
	resultDb := buildCondition(q, opts...)
	var results []*R
	resultDb.Scopes(paginate(page)).Scan(&results)
	page.Records = results
	return page, resultDb
}

// SelectPageMaps 根据条件分页查询，返回分页Map记录
func SelectPageMaps[T any](page *Page[map[string]any], q *Query[T], opts ...OptionFunc) (*Page[map[string]any], *gorm.DB) {
	total, countDb := SelectCount[T](q, opts...)
	if countDb.Error != nil {
		return page, countDb
	}
	page.Total = total
	resultDb := buildCondition(q, opts...)
	var results []map[string]any
	resultDb.Scopes(paginate(page)).Find(&results)
	for _, m := range results {
		page.Records = append(page.Records, &m)
	}
	return page, resultDb
}

// SelectCount 根据条件查询记录数量
func SelectCount[T any](q *Query[T], opts ...OptionFunc) (int64, *gorm.DB) {
	var count int64
	resultDb := buildCondition(q, opts...)
	resultDb.Count(&count)
	return count, resultDb
}

func paginate[T any](p *Page[T]) func(db *gorm.DB) *gorm.DB {
	page := p.Current
	pageSize := p.Size
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func buildCondition[T any](q *Query[T], opts ...OptionFunc) *gorm.DB {
	db := getDb(opts...)
	resultDb := db.Model(new(T))
	if q != nil {
		if len(q.DistinctColumns) > 0 {
			resultDb.Distinct(q.DistinctColumns)
		}

		if len(q.SelectColumns) > 0 {
			resultDb.Select(q.SelectColumns)
		}

		if q.QueryBuilder.Len() > 0 {

			if q.AndBracketBuilder.Len() > 0 {
				q.QueryArgs = append(q.QueryArgs, q.AndBracketArgs...)
				q.QueryBuilder.WriteString(q.AndBracketBuilder.String())
			}

			if q.OrBracketBuilder.Len() > 0 {
				q.QueryArgs = append(q.QueryArgs, q.OrBracketArgs...)
				q.QueryBuilder.WriteString(q.OrBracketBuilder.String())
			}

			resultDb.Where(q.QueryBuilder.String(), q.QueryArgs...)
		}

		if len(q.ConditionMap) > 0 {
			var condMap = make(map[string]any)
			for k, v := range q.ConditionMap {
				columnName := getColumnName(k)
				condMap[columnName] = v
			}
			resultDb.Where(condMap)
		}

		if q.OrderBuilder.Len() > 0 {
			resultDb.Order(q.OrderBuilder.String())
		}

		if q.GroupBuilder.Len() > 0 {
			resultDb.Group(q.GroupBuilder.String())
		}

		if q.HavingBuilder.Len() > 0 {
			resultDb.Having(q.HavingBuilder.String(), q.HavingArgs...)
		}
	}
	return resultDb
}

func getPkColumnName[T any]() string {
	var entity T
	entityType := reflect.TypeOf(entity)
	numField := entityType.NumField()
	var columnName string
	for i := 0; i < numField; i++ {
		field := entityType.Field(i)
		tagSetting := schema.ParseTagSetting(field.Tag.Get("gorm"), ";")
		isPrimaryKey := utils.CheckTruth(tagSetting["PRIMARYKEY"], tagSetting["PRIMARY_KEY"])
		if isPrimaryKey {
			name, ok := tagSetting["COLUMN"]
			if !ok {
				namingStrategy := schema.NamingStrategy{}
				name = namingStrategy.ColumnName("", field.Name)
			}
			columnName = name
			break
		}
	}
	if columnName == "" {
		return constants.DefaultPrimaryName
	}
	return columnName
}

func getDb(opts ...OptionFunc) *gorm.DB {
	var config Option
	for _, op := range opts {
		op(&config)
	}

	// Clauses()目的是为了初始化Db，如果db已经被初始化了,会直接返回db
	var db = globalDb.Clauses()

	if config.Db != nil {
		db = config.Db.Clauses()
	}

	// 设置需要忽略的字段
	setOmitIfNeed(config, db)

	// 设置选择的字段
	setSelectIfNeed(config, db)

	return db
}

func setSelectIfNeed(config Option, db *gorm.DB) {
	if len(config.Selects) > 0 {
		var columnNames []string
		for _, column := range config.Selects {
			columnName := getColumnName(column)
			columnNames = append(columnNames, columnName)
		}
		db.Select(columnNames)
	}
}

func setOmitIfNeed(config Option, db *gorm.DB) {
	if len(config.Omits) > 0 {
		var columnNames []string
		for _, column := range config.Omits {
			columnName := getColumnName(column)
			columnNames = append(columnNames, columnName)
		}
		db.Omit(columnNames...)
	}
}

func Begin(opts ...*sql.TxOptions) *gorm.DB {
	db := getDb()
	return db.Begin(opts...)
}