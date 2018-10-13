package common

type Directive struct {
	Name Ident
	Args ArgumentList
}

func ParseDirectives(l *Lexer) DirectiveList {
	var directives DirectiveList
	for l.Peek() == '@' {
		l.ConsumeToken('@')
		d := &Directive{}
		d.Name = l.ConsumeIdentWithLoc()
		d.Name.Loc.Column--
		if l.Peek() == '(' {
			d.Args = ParseArguments(l)
		}
		directives = append(directives, d)
	}
	return directives
}

type DirectiveList []*Directive

func (l DirectiveList) Get(name string) *Directive {
	for _, d := range l {
		if d.Name.Name == name {
			return d
		}
	}
	return nil
}
