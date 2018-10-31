package database

/////////////////////////////////////////////////////// Session  //////////////////////////////////////////////////////////
type Session struct {
	db       DataBase
	apply    bool
	reversion int64
}

//func (session *Session) Commit(reversion int64) {
//	if !session.apply {
//		// log ?
//		return
//	}
//	session.db.Commit(reversion)
//	session.apply = false
//}

func (session *Session) Push() {
	if session.db == nil {
		return
	}
	session.apply = false
	session.db = nil
}
func (session *Session) Squash() {
	if session.db == nil || !session.apply {
		return
	}

	session.db.squash()
	session.db = nil
	session.apply = false
}

func (session *Session) Undo() {
	if session.db == nil || !session.apply {
		return
	}

	session.db.Undo()
	session.db = nil
	session.apply = false
}

func (session *Session) Revision() int64 {
	return session.reversion
}
