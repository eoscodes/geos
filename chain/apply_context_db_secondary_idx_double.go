package chain

import (
	//"github.com/eosspark/eos-go/chain/types"
	"github.com/eosspark/eos-go/entity"
)

type IdxDouble struct {
	context  *ApplyContext
	itrCache *iteratorCache
}

func NewIdxDouble(c *ApplyContext) *IdxDouble {
	return &IdxDouble{
		context:  c,
		itrCache: NewIteratorCache(),
	}
}

func (i *IdxDouble) store(scope int64, table int64, payer int64, id int64, secondary *entity.Float64_t) int {
	return 0
	// //EOS_ASSERT( common.AccountName(payer) != common.AccountName{}, invalid_table_payer, "must specify a valid account to pay for new record" );
	// tab := i.context.FindOrCreateTable(int64(i.context.Receiver), scope, table, payer)

	// obj := &types.SecondaryObjectDouble{
	// 	TId:          tab.ID,
	// 	PrimaryKey:   id,
	// 	SecondaryKey: *secondary,
	// 	Payer:        payer,
	// }

	// i.context.DB.Insert(obj)
	// i.context.DB.Modify(tab, func(t *types.TableIdObject) {
	// 	t.Count++
	// })

	// i.context.UpdateDbUsage(payer, types.BillableSizeV(obj.GetBillableSizeValue()))

	// i.itrCache.cacheTable(tab)
	// return i.itrCache.add(obj)
}

func (i *IdxDouble) remove(iterator int) int {
	return 0

	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))
	// i.context.UpdateDbUsage(obj.payer, - types.BillableSizeV(obj.GetBillableSize())

	// tab := i.itrCache.getTable(obj.TId)
	// i.context.DB.Modify(tab, func(t *types.TableIdObject) {
	// 	t.Count--
	// })

	// i.context.DB.Remove(obj)
	// if tab.Count == 0 {
	// 	i.context.Remove(tab)
	// }
	// i.itrCache.remove(iterator)
}

func (i *IdxDouble) update(iterator int, payer int64, secondary *entity.Float64_t) {

	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))
	// objTable := i.itrCache.getTable(obj.TId)

	// //EOS_ASSERT( table_obj.code == i.context.Receiver, table_access_violation, "db access violation" )
	// if payer == common.AccountName{} {payer = obj.Payer}

	// billingSize := obj.GetBillableSize()
	// if obj.Payer != payer {
	// 	i.context.UpdateDbUsage(obj.Payer, - types.BillableSizeV(billingSize))
	// 	i.context.UpdateDbUsage(payer, + types.BillableSizeV(billingSize))
	// }

	// i.context.DB.Modify(obj, func(o *types.SecondaryObjectDouble){
	// 	o.SecondaryKey = *secondary
	// 	o.Payer = payer
	// })
}

func (i *IdxDouble) findSecondary(code int64, scope int64, table int64, secondary *entity.Float64_t, primary *uint64) int {
	return 0
	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {return -1}

	// tableEndItr := i.itrCache.cacheTable(&tab)

	// obj := &types.SecondaryObjectDouble{}
	// err := i.context.DB.get("bySecondary", obj, obj.MakeTuple(tab.ID, *secondary)

	// *primary = obj.PrimaryKey

	// if err == nil {return tableEndItr}
	// return i.itrCache.add(obj)
}

func (i *IdxDouble) lowerbound(code int64, scope int64, table int64, secondary *entity.Float64_t, primary *uint64) int {
	return 0
	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }

	// tableEndItr := i.itrCache.cacheTable(tab)

	// obj := types.SecondaryObjectDouble{}

	// idx := i.context.DB.GetIndex("bySecondary", &obj)
	// itr := idx.Lowerbound(obj.maketuple(tab.ID, *secondary))
	// if itr == idx.End() {return tableEndItr}

	// objLowerbound := (*types.SecondaryObjectDouble)(itr.GetObject())
	// if objLowerbound.TId != tab.ID {return tableEndItr}

	// *primary = objLowerbound.PrimaryKey
	// *secondary = objLowerbound.SecondaryKey

	// return i.itrCache.add(objLowerbound)
}

func (i *IdxDouble) upperbound(code int64, scope int64, table int64, secondary *entity.Float64_t, primary *uint64) int {
	return 0
	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }

	// tableEndItr := i.itrCache.cacheTable(tab)

	// obj := &types.SecondaryObjectDouble{}

	// idx := i.context.DB.GetIndex("bySecondary", obj)
	// itr := idx.Upperbound(obj.maketuple(tab.ID, *secondary))
	// if itr == idx.End() {
	// 	return tableEndItr
	// }

	// objUpperbound = (*types.SecondaryObjectDouble)(itr.GetObject())
	// if objUpperbound.TId != tab.ID {
	// 	return tableEndItr
	// }

	// *primary = objUpperbound.PrimaryKey
	// *secondary = objUpperbound.SecondaryKey

	// return i.itrCache.add(objUpperbound)
}

func (i *IdxDouble) end(code int64, scope int64, table int64) int {
	return 0

	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }
	// return i.itrCache.cacheTable(tab)
}

func (i *IdxDouble) next(iterator int, primary *uint64) int {
	return 0

	// if iterator < -1 {
	// 	return -1
	// }
	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))

	// idx := i.context.DB.GetIndex("bySecondary", obj)
	// itr := idx.IteratorTo(obj)

	// itrNext := itr.Next()
	// objNext := (*types.SecondaryObject)(itrNext.GetObject())

	// if itr == idx.End() || objNext.TId != obj.TId {
	// 	return i.itrCache.getEndIteratorByTableID(obj.TId)
	// }

	// *primary = objNext.PrimaryKey
	// return i.itrCache.add(objNext)

}

func (i *IdxDouble) previous(iterator int, primary *uint64) int {
	return 0

	// idx := i.context.DB.GetIndex("bySecondary", &types.SecondaryObjectDouble{})

	// if( iterator < -1) {
	//     tab = i.itrCache.findTablebyEndIterator(iterator)
	//    //EOS_ASSERT( tab, invalid_table_iterator, "not a valid end iterator" );

	//    itr := idx.Upperbound(tab.ID)
	//    if( idx.begin() == idx.end() || itr == idx.begin() ) return -1;

	//    itrPrev := itr.Prev()
	//    objPrev := itr.GetObject()
	//    if( objPrev.TId != tab->ID ) return -1;

	//    *primary = objPrev.PrimaryKey
	//    return a.KeyvalCache.add(objPrev)
	// }

	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))
	// itr := idx.IteratorTo(obj)

	// if itr == idx.Begin() {
	// 	return -1
	// }
	// itrNext := itr.Prev()
	// objNext := (*types.SecondaryObjectDouble)(itr.GetObject())

	// if objNext.TId != obj.TId {
	// 	return -1
	// }
	// *primary = objNext.PrimaryKey
	// return i.itrCache.add(objNext)
}

func (i *IdxDouble) findPrimary(code int64, scope int64, table int64, secondary *entity.Float64_t, primary *uint64) int {
	return 0

	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }

	// tableEndItr := i.itrCache.cacheTable(tab)

	// obj := &types.SecondaryObjectDouble{}
	// err := i.context.DB.get("byPrimary", &obj, ObjectType.makeTuple(tab.ID, *primary))

	// *secondary = obj.SecondaryKey

	// if err == nil {
	// 	return tableEndItr
	// }
	// return i.itrCache.add(obj)
}

func (i *IdxDouble) lowerboundPrimary(code int64, scope int64, table int64, primary *uint64) int {
	return 0

	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }

	// tableEndItr := i.itrCache.cacheTable(tab)

	// obj := &types.SecondaryObjectDouble{}
	// idx := i.context.DB.GetIndex("byPrimary",obj)
	// itr := idex.Lowerbound(obj.MakeTuple(tab.ID, *primary))
	// if itr == idx.End() {return tableEndItr}

	// objLowerbound := (*types.SecondaryObjectDouble)(itr.GetObject())
	// if objLowerbound.TId != tab.ID {return tableEndItr}

	// return i.itrCache.add(objLowerbound)
}

func (i *IdxDouble) upperboundPrimary(code int64, scope int64, table int64, primary *uint64) int {
	return 0

	// tab := i.context.FindTable(code, scope, table)
	// if tab == nil {
	// 	return -1
	// }

	// tableEndItr := i.itrCache.cacheTable(tab)

	//    obj := &types.SecondaryObjectDouble{}
	// idx := i.context.DB.GetIndex("byPrimary",  obj)
	// itr := idx.UpperBound(obj.MakeTuple(tab.ID, *primary))
	// if itr == idx.End() {
	// 	return tableEndItr
	// }

	// objUpperbound := (*types.SecondaryObjectDouble)(itr.GetObject())
	// if obj.TId != tab.ID {
	// 	return tableEndItr
	// }

	//    i.itrCache.cacheTable(tab)
	// return i.itrCache.add(objUpperbound)
}

func (i *IdxDouble) nextPrimary(iterator int, primary *uint64) int {
	return 0

	// if iterator < -1 {
	// 	return -1
	// }
	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))
	// idx := i.context.DB.GetIndex("byPrimary",  &types.SecondaryObjectDouble{})

	// itr := idx.iteratorTo(obj)

	// itrNext := itr.Next()
	// objNext := (*types.SecondaryObjectDouble)(itrNext.GetObject())

	// if itr == idx.End() || objNext.TId != obj.TId {
	// 	return i.itrCache.getEndIteratorByTableID(obj.TId)
	// }

	// *primary = objNext.PrimaryKey
	// return i.itrCache.add(objNext)

}

func (i *IdxDouble) previousPrimary(iterator int, primary *uint64) int {
	return 0

	// idx := i.context.DB.GetIndex("byPrimary", &types.SecondaryObjectDouble{})

	// if( iterator < -1) {
	//     tab = i.itrCache.findTablebyEndIterator(iterator)
	//    //EOS_ASSERT( tab, invalid_table_iterator, "not a valid end iterator" );

	//    itr := idx.Upperbound(tab.ID)
	//    if idx.begin() == idx.end() || itr == idx.begin() {return -1}

	//    itrPrev := itr.Prev()
	//    objPrev := (*types.SecondaryObjectDouble)(itr.GetObject())
	//    if objPrev.TId != tab->ID { return -1}

	//    *primary = objPrev.PrimaryKey
	//    return a.KeyvalCache.add(objPrev)
	// }

	// obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))
	// itr := idx.IteratorTo(obj)

	// if itr == idx.Begin() {
	// 	return -1
	// }
	// itrNext := itr.Prev()
	// objNext := (*types.SecondaryObjectDouble)(itr.GetObject())

	// if objNext.TId != obj.TId {
	// 	return -1
	// }
	// *primary = objNext.PrimaryKey
	// return i.itrCache.add(objNext)
}

func (i *IdxDouble) get(iterator int, secondary *entity.Float64_t, primary *uint64) {
	//  obj := (*types.SecondaryObjectDouble)(i.itrCache.get(iterator))

	// *secondary = obj.SecondaryKey
	// *primary = obj.PrimaryKey
}
