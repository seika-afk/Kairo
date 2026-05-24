package ot

type Op struct {
	Type     string `json:"type"` //insert or delete
	Position int    `json:"pos"`
	Text     string `json:"text"`
	Length   int    `json:"length"`
	UserID   string `json:"user_id"`
	Version  int    `json:"version"`
}

func opSpan(op Op) int {
	switch op.Type {
	case "insert":
		return len([]rune(op.Text))
	case "delete":
		if op.Length < 0 {
			return 0
		}
		return op.Length
	default:
		return 0
	}
}

func Transform(incoming Op, against Op) Op {
	switch incoming.Type {
	case "insert":
		switch against.Type {
		case "insert":
			if against.Position <= incoming.Position {
				incoming.Position += opSpan(against)
			}
		case "delete":
			if against.Position < incoming.Position {
				incoming.Position -= opSpan(against)
				//if the insert ends up inside/before deleted region-> snap it to start of delete
				if incoming.Position < against.Position {
					incoming.Position = against.Position
				}
			}

		}
	case "delete":
		switch against.Type {
		case "insert":
			if against.Position < incoming.Position {
				incoming.Position -= opSpan(against)

			}
		case "delete":
			if against.Position < incoming.Position {
				incoming.Position -= opSpan(against)
				if incoming.Position < against.Position {
					incoming.Position = against.Position
				}
			}
		}
	}
	return incoming
}

func Apply(doc []rune, op Op) []rune {
	if op.Position < 0 {
		op.Position = 0
	}
	if op.Position > len(doc) {
		op.Position = len(doc)
	}

	switch op.Type {
	case "insert":
		insertRunes := []rune(op.Text)
		newDoc := make([]rune, 0, len(doc)+len(insertRunes))
		newDoc = append(newDoc, doc[:op.Position]...)
		newDoc = append(newDoc, insertRunes...)
		newDoc = append(newDoc, doc[op.Position:]...)
		return newDoc

	case "delete":
		if op.Length < 0 {
			return doc
		}
		end := op.Position + op.Length
		if end > len(doc) {
			end = len(doc)
		}
		newDoc := make([]rune, 0, len(doc)-(end-op.Position))
		newDoc = append(newDoc, doc[:op.Position]...)
		newDoc = append(newDoc, doc[end:]...)
		return newDoc

	}
	return doc
}

func TransformAgainstHistory(op Op, history []Op, since int) Op {

	//iterate across the history , and if its less than since continue
	for _, historyOp := range history[since:] {
		op = Transform(op, historyOp)
	}
	return op
}
