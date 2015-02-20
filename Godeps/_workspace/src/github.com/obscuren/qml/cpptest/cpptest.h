#ifndef UONEAUTH_H
#define UONEAUTH_H

#include <stdint.h>
#include <stddef.h>
#include <stdlib.h>

typedef void TestType_;
typedef void PlainTestType_;

#ifdef __cplusplus
extern "C" {
#endif

TestType_ *newTestType();

int plainTestTypeN(PlainTestType_ *plain);

#ifdef __cplusplus
}
#endif

#endif // UONEAUTH_H
