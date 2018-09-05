package eosiodb

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/eosspark/eos-go/db/storm"
	"path/filepath"
	"reflect"
	"sync"
)

/////////////////////////////////////////////////////// Global Func  //////////////////////////////////////////////////////////
func copyInterface(data interface{}) interface{} {
	src := reflect.ValueOf(data)
	dst := reflect.New(reflect.Indirect(src).Type())

	srcElem := src.Elem()
	dstElem := dst.Elem()
	NumField := srcElem.NumField()
	for i := 0; i < NumField; i++ {
		sf := srcElem.Field(i)
		df := dstElem.Field(i)
		df.Set(sf)
	}
	return dst.Interface()
}

func equal(m map[interface{}]uint64, data interface{}) interface{} {
	for key, value := range m {
		if reflect.DeepEqual(key, data) {
			fmt.Println(value)
			return key
		}
	}
	return nil
}

func remove(m map[interface{}]uint64, data interface{}) {
	key := equal(m, data)
	delete(m, key)
}

/////////////////////////////////////////////////////// UndoState  //////////////////////////////////////////////////////////
type undoState struct {
	NewValue    map[interface{}]uint64 // hash or md5 ?
	RemoveValue map[interface{}]uint64
	OldValue    map[interface{}]uint64
	version     uint64
}

func newUndoState(version uint64) *undoState {
	return &undoState{
		NewValue:    make(map[interface{}]uint64),
		RemoveValue: make(map[interface{}]uint64),
		OldValue:    make(map[interface{}]uint64),
		version:     version,
	}
}

func (stack *undoState) insert(data interface{}) {
	stack.NewValue[data] = stack.version
}

func (stack *undoState) remove(data interface{}) {
	_, ok := stack.NewValue[data]
	if ok {
		remove(stack.NewValue, data)
		return
	}
	_, ok = stack.OldValue[data]
	if ok {
		stack.RemoveValue[data] = stack.version
		remove(stack.OldValue, data)
		return
	}

	_, ok = stack.RemoveValue[data]
	if ok {
		return
	}
	stack.RemoveValue[data] = stack.version
}

func (stack *undoState) update(data interface{}) {
	key := equal(stack.NewValue, data)
	if key != nil {
		return
	}
	key = equal(stack.OldValue, data)
	if key != nil {
		return
	}
	stack.version++
	stack.OldValue[data] = stack.version
}

/////////////////////////////////////////////////////// Database  //////////////////////////////////////////////////////////
type base struct {
	db   *storm.DB
	path string
	file string
	rw   bool // XXX read only or read write
}

func (db *base) checkState() error {
	if !db.rw {
		return errors.New("read only")
	}
	return nil
}

func (db *base) insert(data interface{}) error {
	err := db.checkState()
	if err != nil {
		return err
	}
	return db.db.Save(data)
}

func (db *base) remove(data interface{}) error {
	err := db.checkState()
	if err != nil {
		return err
	}
	return db.db.DeleteStruct(data) // 	db.db.DeleteStruct ?
}

func (db *base) updateItem(old interface{}) error {
	return db.db.Update(old)
}

func (db *base) update(old interface{}, fn func(interface{}) error) error {
	err := db.checkState()
	if err != nil {
		return err
	}

	err = fn(old)
	if err != nil {
		return err
	}

	return db.updateItem(old)
}

func (db *base) byIndex(fieldName string, to interface{}) error {
	return db.db.AllByIndex(fieldName, to)
}

func (db *base) all(data interface{}) error {
	return db.db.All(data)
}

func (db *base) find(fieldName string, value interface{}, to interface{}) error {
	return db.db.One(fieldName, value, to)
}

func (db *base) get(fieldName string, fieldValue interface{}, to interface{}) error {
	return db.db.Find(fieldName, fieldValue, to)
}

func (db *base) updateField(data interface{}, fieldName string, value interface{}) error {
	err := db.checkState()
	if err != nil {
		return err
	}
	return db.db.UpdateField(data, fieldName, value)
}

func (db *base) close() {
	db.db.Close()
}

func newBase(path string, name string, rw bool /*read and  write*/) (*base, error) {
	dir := filepath.Join(path, name)
	fmt.Println("database dir : ", dir)
	db, err := storm.Open(name)
	if err != nil {
		return nil, err
	}
	return &base{
		db:   db,
		path: path,
		file: name,
		rw:   rw,
	}, nil
}

/////////////////////////////////////////////////////// DataBase  //////////////////////////////////////////////////////////
type DataBase struct {
	Db      *base
	Stack   *deque
	Version uint64
	Flag    bool
}

func NewDataBase(path string, name string, rw bool) (*DataBase, error) {
	db, err := newBase(path, name, rw)
	if err != nil {
		return nil, err
	}
	return &DataBase{Db: db, Stack: newDeque()}, nil
}

func (undo *DataBase) Close() {
	undo.Db.close()
	if undo.Flag {
		undo.undo()
	}
}

func (undo *DataBase) commit(version uint64) {
	if !undo.Flag {
		return
	}
	for {
		stack := undo.getFirstStack()
		if stack == nil {
			break
		}
		if stack.version <= version {
			undo.Stack.PopFront()
			undo.Version--
		} else {
			break
		}
	}
}

func (undo *DataBase) squash() {
	if !undo.Flag {
		return
	}
	if undo.Stack.Size() == 1 {
		undo.Stack.Pop()
		undo.Version--
		return
	}
	stack := undo.getStack()
	prestack := undo.getSecond()
	for key, value := range stack.OldValue {
		if _, ok := prestack.NewValue[key]; ok {
			continue
		}
		if _, ok := prestack.OldValue[key]; ok {
			continue
		}
		if _, ok := prestack.RemoveValue[key]; ok {
			fmt.Println("squash failed")
			// panic ?
		}
		prestack.OldValue[key] = value
	}

	for key, value := range stack.NewValue {
		prestack.NewValue[key] = value
	}

	for key, value := range stack.RemoveValue {
		fmt.Println(key, " --> ", value)
		if _, ok := prestack.NewValue[key]; ok {
			k := equal(prestack.NewValue, key)
			delete(prestack.NewValue, k)
		}
		if _, ok := prestack.OldValue[key]; ok {
			prestack.RemoveValue[key] = value
			k := equal(prestack.OldValue, key)
			delete(prestack.OldValue, k)
		}
		prestack.RemoveValue[key] = value
	}
	undo.Stack.Pop()
	undo.Version--
}

func (undo *DataBase) undo() {
	if !undo.Flag {
		return
	}
	stack := undo.getStack()
	if stack == nil {
		return
	}
	for key, _ := range stack.OldValue {
		undo.Db.updateItem(key)
	}
	for key, _ := range stack.NewValue {
		undo.Db.remove(key)
	}
	for key, _ := range stack.RemoveValue {
		undo.Db.insert(key)
	}
	undo.Stack.Pop()
	undo.Version--
}

func (undo *DataBase) startSession() *Session {
	undo.Version++
	undo.Flag = true
	state := newUndoState(undo.Version)
	undo.Stack.Append(state)
	return &Session{Db: undo, apply: true, version: undo.Version}
}

func (undo *DataBase) getFirstStack() *undoState {
	stack := undo.Stack.First()
	switch typ := stack.(type) {
	case *undoState:
		return typ
	default:
		return nil
		//panic(TYPE_NOT_FOUND)
	}
	return nil
}

func (undo *DataBase) getSecond() *undoState {
	stack := undo.Stack.LastSecond()
	switch typ := stack.(type) {
	case *undoState:
		return typ
	default:
		return nil
		//panic(TYPE_NOT_FOUND)
	}
	return nil

}

func (undo *DataBase) getStack() *undoState {
	stack := undo.Stack.Last()
	if stack == nil {
		return nil
	}
	switch typ := stack.(type) {
	case *undoState:
		return typ
	default:
		return nil
		//panic(TYPE_NOT_FOUND)
	}
	return nil
}

func (undo *DataBase) Insert(data interface{}) error {
	err := undo.Db.insert(data)
	if err != nil {
		return err
	}
	if !undo.Flag {
		return nil
	}
	stack := undo.getStack()
	if stack == nil {
		return errors.New("undo session empty")
	}
	copy := copyInterface(data)
	stack.insert(copy)
	return nil
}

func (undo *DataBase) Remove(data interface{}) error {
	err := undo.Db.remove(data)
	if err != nil {
		return err
	}
	if !undo.Flag {
		return nil
	}
	stack := undo.getStack()
	if stack == nil {
		return errors.New("undo session empty")
	}
	copy := copyInterface(data)
	stack.remove(copy)
	return nil
}

func (undo *DataBase) Update(old interface{}, fn func(interface{}) error) error {
	copy := copyInterface(old)
	err := undo.Db.update(old, fn)
	if err != nil {
		return err
	}
	if !undo.Flag {
		return nil
	}
	stack := undo.getStack()
	if stack == nil {
		return errors.New("undo session empty")
	}
	stack.update(copy)
	return nil
}

func (undo *DataBase) All(data interface{}) error {
	return undo.Db.all(data)
}

func (undo *DataBase) Find(fieldName string, value interface{}, to interface{}) error {
	return undo.Db.find(fieldName, value, to)
}

func (undo *DataBase) Get(fieldName string, fieldValue interface{}, to interface{}) error {
	return undo.Db.get(fieldName, fieldValue, to)
}

func (undo *DataBase) ByIndex(fieldName string, to interface{}) error {
	return undo.Db.byIndex(fieldName, to)
}

/////////////////////////////////////////////////////// Session  //////////////////////////////////////////////////////////
type Session struct {
	Db      *DataBase
	version uint64
	apply   bool
}

func (session *Session) Commit() {
	if !session.apply {
		// log ?
		return
	}
	version := session.version
	session.Db.commit(version)
	session.apply = false
}

func (session *Session) Squash() {
	if !session.apply {
		return
	}
	session.Db.squash()
	session.apply = false
}

func (session *Session) Undo() {
	if !session.apply {
		return
	}
	session.Db.undo()
	session.apply = false
}

/////////////////////////////////////////////////////// Deque //////////////////////////////////////////////////////////
type deque struct {
	sync.RWMutex
	container *list.List
	capacity  int
}

func newDeque() *deque {
	return newCappedDeque(-1)
}

func newCappedDeque(capacity int) *deque {
	return &deque{
		container: list.New(),
		capacity:  capacity,
	}
}

func (s *deque) Append(item interface{}) bool {
	s.Lock()
	defer s.Unlock()

	if s.capacity < 0 || s.container.Len() < s.capacity {
		s.container.PushBack(item)
		return true
	}
	return false
}

func (s *deque) PopFront() interface{} {
	s.Lock()
	defer s.Unlock()

	var item interface{} = nil
	var firstContainerItem *list.Element = nil

	firstContainerItem = s.container.Front()
	if firstContainerItem != nil {
		item = s.container.Remove(firstContainerItem)
	}

	return item
}

func (s *deque) LastSecond() interface{} {
	last := s.PopFront()
	second := s.PopFront()
	s.Append(second)
	s.Append(last)
	return second
}

func (s *deque) Pop() interface{} {
	s.Lock()
	defer s.Unlock()

	var item interface{} = nil
	var lastContainerItem *list.Element = nil

	lastContainerItem = s.container.Back()
	//two := s.container[2]
	if lastContainerItem != nil {
		item = s.container.Remove(lastContainerItem)
	}

	return item
}

func (s *deque) Size() int {
	s.RLock()
	defer s.RUnlock()

	return s.container.Len()
}

func (s *deque) First() interface{} {
	s.RLock()
	defer s.RUnlock()

	item := s.container.Front()
	if item != nil {
		return item.Value
	} else {
		return nil
	}
}

func (s *deque) Last() interface{} {
	s.RLock()
	defer s.RUnlock()

	item := s.container.Back()
	if item != nil {
		return item.Value
	} else {
		return nil
	}
}

func (s *deque) Empty() bool {
	s.RLock()
	defer s.RUnlock()

	return s.container.Len() == 0
}

/////////////////////////////////////////////////////// Simple Test //////////////////////////////////////////////////////////

type Item struct {
	ID   int    `storm:"id,increment"`
	Name string `storm:"index"`
	Tag  int    `storm:"index"`
}

func Write() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()

	userNames := []string{"lx", "li", "elk", "fox", "lion"}
	for _, name := range userNames {
		it := Item{Name: name, Tag: 100}
		db.Insert(&it)
	}
}

func GetAll() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()
	var items []Item
	db.All(&items)
	fmt.Println(items)
}

func Remove() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()

	it := Item{ID: 1, Name: "lx", Tag: 100}
	err = db.Remove(&it)
	if err != nil {
		fmt.Println(err)
	}
}

func Update() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()

	fn := func(data interface{}) error {
		ref := reflect.ValueOf(data).Elem()
		if ref.CanSet() {
			ref.Field(1).SetString("linx")
			ref.Field(2).SetInt(110)
		} else {
			fmt.Println("ref can not set")
			// error log ?
		}
		return nil
	}
	it := Item{ID: 2, Name: "fox", Tag: 100}
	err = db.Update(&it, fn)
	if err != nil {
		fmt.Println("updata failed")
	}
}

func UndoSession() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()
	session := db.startSession()
	defer session.Undo()
	it := Item{Name: "qieqie", Tag: 190}
	db.Insert(&it)
	var items []Item
	db.All(&items)
	fmt.Println(items)
}

func MultiSession() {
	db, err := NewDataBase("./", "test", true)
	if err != nil {
		fmt.Println("NewDatabase failed")
	}
	defer db.Close()
	session := db.startSession()
	defer session.Undo()
	it := Item{Name: "qieqie", Tag: 190}
	db.Insert(&it)

	var items []Item
	db.All(&items)
	fmt.Println(items)

	session2 := db.startSession()
	defer session2.Undo()

	it2 := Item{Name: "garytone", Tag: 1088}
	db.Insert(&it2)

	var items2 []Item
	db.All(&items2)
	fmt.Println(items2)

	session2.Squash()
	//session2.Commit()

	var itemAll []Item
	db.All(&itemAll)
	fmt.Println(itemAll)
	session.Commit()
}

func main() {
	Write()
	GetAll()
	Remove()
	GetAll()
	Update()
	GetAll()
	fmt.Println("------------ undo --------------")
	UndoSession()
	GetAll()
	fmt.Println("------------ MultiSession --------------")
	MultiSession()
	GetAll()
}
