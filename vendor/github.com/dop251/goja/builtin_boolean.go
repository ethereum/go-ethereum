package goja

func (r *Runtime) booleanproto_toString(call FunctionCall) Value {
	var b bool
	switch o := call.This.(type) {
	case valueBool:
		b = bool(o)
		goto success
	case *Object:
		if p, ok := o.self.(*primitiveValueObject); ok {
			if b1, ok := p.pValue.(valueBool); ok {
				b = bool(b1)
				goto success
			}
		}
	}
	r.typeErrorResult(true, "Method Boolean.prototype.toString is called on incompatible receiver")

success:
	if b {
		return stringTrue
	}
	return stringFalse
}

func (r *Runtime) booleanproto_valueOf(call FunctionCall) Value {
	switch o := call.This.(type) {
	case valueBool:
		return o
	case *Object:
		if p, ok := o.self.(*primitiveValueObject); ok {
			if b, ok := p.pValue.(valueBool); ok {
				return b
			}
		}
	}

	r.typeErrorResult(true, "Method Boolean.prototype.valueOf is called on incompatible receiver")
	return nil
}

func (r *Runtime) initBoolean() {
	r.global.BooleanPrototype = r.newPrimitiveObject(valueFalse, r.global.ObjectPrototype, classBoolean)
	o := r.global.BooleanPrototype.self
	o._putProp("toString", r.newNativeFunc(r.booleanproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.booleanproto_valueOf, nil, "valueOf", nil, 0), true, false, true)

	r.global.Boolean = r.newNativeFunc(r.builtin_Boolean, r.builtin_newBoolean, "Boolean", r.global.BooleanPrototype, 1)
	r.addToGlobal("Boolean", r.global.Boolean)
}
