#include <string.h>

#include "cpptest.h"
#include "testtype.h"

TestType_ *newTestType()
{
	return new TestType();
}

int plainTestTypeN(PlainTestType_ *plain)
{
	return static_cast<PlainTestType *>(plain)->n;
}
