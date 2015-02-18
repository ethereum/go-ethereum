#include <QObject>

#include "connector.h"
#include "capi.h"

Connector::~Connector()
{
    hookSignalDisconnect(func);
}

void Connector::invoke()
{
    panicf("should never get called");
}

int Connector::qt_metacall(QMetaObject::Call c, int idx, void **a)
{
    if (c == QMetaObject::InvokeMetaMethod && idx == metaObject()->methodOffset()) {
        DataValue args[MaxParams];
        QObject *plain = NULL;
        for (int i = 0; i < argsLen; i++) {
            int paramType = method.parameterType(i);
            if (paramType == 0 && a[1 + i] != NULL) {
                const char *typeName = method.parameterTypes()[i].constData();
                void *addr = a[1 + i];
                if (typeName[strlen(typeName)-1] == '*') {
                    addr = *(void **)addr;
                }
                plain = new PlainObject(typeName, addr, plain);
                QVariant var = QVariant::fromValue((QObject *)plain);
                packDataValue(&var, &args[i]);
            } else {
                QVariant var(method.parameterType(i), a[1 + i]);
                packDataValue(&var, &args[i]);
            }
        }
        hookSignalCall(engine, func, args);
        if (plain != NULL) {
                delete plain;
        }
        return -1;
    }
    return standard_qt_metacall(c, idx, a);
}

// vim:ts=4:sw=4:et
