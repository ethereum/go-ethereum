#ifndef CONNECTOR_H
#define CONNECTOR_H

#include <QObject>

#include <stdint.h>

class Connector : public QObject
{
    Q_OBJECT

    public:

    Connector(QObject *sender, QMetaMethod method, QQmlEngine *engine, void *func, int argsLen)
        : QObject(sender), engine(engine), method(method), func(func), argsLen(argsLen) {};

    virtual ~Connector();

    // MOC HACK: s/Connector::qt_metacall/Connector::standard_qt_metacall/
    int standard_qt_metacall(QMetaObject::Call c, int idx, void **a);

    public slots:

    void invoke();

    private:

    QQmlEngine *engine;
    QMetaMethod method;
    void *func;
    int argsLen;
};

class PlainObject : public QObject
{
    Q_OBJECT

    Q_PROPERTY(QString plainType READ getPlainType)
    Q_PROPERTY(void *plainAddr READ getPlainAddr)

    QString plainType;
    void *plainAddr;

    public:

    PlainObject(QObject *parent = 0)
        : QObject(parent) {};

    PlainObject(const char *plainType, void *plainAddr, QObject *parent = 0)
        : QObject(parent), plainType(plainType), plainAddr(plainAddr) {};

    QString getPlainType() { return plainType; };
    void *getPlainAddr() { return plainAddr; };
};

#endif // CONNECTOR_H

// vim:ts=4:sw=4:et
