package ot


type Op struct {
	Type     string `json:"type"` //insert or delete
	Position int `json:"pos"`
	Text     string `json:"text"`
	Length   int    `json:"length"`
	UserID   string `json:"user_id"`
	Version  int    `json:"version"`
}

func transform(incoming Op, against Op) Op {
	switch incoming.Type {
	case "insert":
		switch against.Type {
		case "insert":
			if against.Position <= incoming.Position {
				incoming.Position += len([]rune(against.Text))
			}
		case "delete":
		If delete occurred BEFORE incoming position:
			if against.Position< incoming.Position{
				incoming.Position-= against.Length
				//if the insert ends up inside/before deleted region-> snap it to start of delete
				if incoming.Position< against.Position{
					incoming.Position = against.Position
				}
			}

		}
	case "delete":
		switch against.Type {
		case "insert":
				if against.Position <incoming.Position{
					incoming.Position-= against.Length

				}
		case "delete":
				if against.Position<incoming.Position {
					incoming.Position-=against.Length
					if incoming.Position < against.Position {
										incoming.Position = against.Position
									}
				}
		}
	}

}
