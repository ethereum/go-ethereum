package filter

type Generic struct {
	Str1, Str2, Str3 string

	Fn func(data interface{})
}

func (self Generic) Compare(f Filter) bool {
	filter := f.(Generic)
	if (len(self.Str1) == 0 || filter.Str1 == self.Str1) &&
		(len(self.Str2) == 0 || filter.Str2 == self.Str2) &&
		(len(self.Str3) == 0 || filter.Str3 == self.Str3) {
		return true
	}

	return false
}

func (self Generic) Trigger(data interface{}) {
	self.Fn(data)
}
