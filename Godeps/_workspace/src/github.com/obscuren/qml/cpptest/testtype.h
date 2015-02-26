#ifndef TESTTYPE_H
#define TESTTYPE_H

#include <QObject>

class PlainTestType {

    public:

    PlainTestType(int n) : n(n) {};

    int n;
};

class TestType : public QObject
{
    Q_OBJECT

    Q_PROPERTY(void *voidAddr READ getVoidAddr)

    void *voidAddr;

    public:

    TestType(QObject *parent = 0) : QObject(parent), voidAddr((void*)42) {};

    void *getVoidAddr() { return voidAddr; };

    Q_INVOKABLE void emitPlain() {
            PlainTestType plain = PlainTestType(42);
            emit plainEmittedCpy(plain);
            emit plainEmittedRef(plain);
            emit plainEmittedPtr(&plain);
    };

    signals:

    void plainEmittedCpy(const PlainTestType plain);
    void plainEmittedRef(const PlainTestType &plain);
    void plainEmittedPtr(const PlainTestType *plain);
};

#endif // TESTTYPE_H

// vim:ts=4:sw=4:et
