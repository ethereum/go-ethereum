#include <QApplication>
#include <QQuickView>
#include <QQuickItem>
#include <QtQml>
#include <QDebug>
#include <QQuickImageProvider>

#include <string.h>

#include "govalue.h"
#include "govaluetype.h"
#include "connector.h"
#include "capi.h"

static char *local_strdup(const char *str)
{
    char *strcopy = 0;
    if (str) {
        size_t len = strlen(str) + 1;
        strcopy = (char *)malloc(len);
        memcpy(strcopy, str, len);
    }
    return strcopy;
}

error *errorf(const char *format, ...)
{
    va_list ap;
    va_start(ap, format);
    QString str = QString().vsprintf(format, ap);
    va_end(ap);
    QByteArray ba = str.toUtf8();
    return local_strdup(ba.constData());
}

void panicf(const char *format, ...)
{
    va_list ap;
    va_start(ap, format);
    QString str = QString().vsprintf(format, ap);
    va_end(ap);
    QByteArray ba = str.toUtf8();
    hookPanic(local_strdup(ba.constData()));
}

void newGuiApplication()
{
    static char empty[1] = {0};
    static char *argv[] = {empty, 0};
    static int argc = 1;
    new QApplication(argc, argv);

    // The event loop should never die.
    qApp->setQuitOnLastWindowClosed(false);
}

void applicationExec()
{
    qApp->exec();
}

void applicationExit()
{
    qApp->exit(0);
}

void applicationFlushAll()
{
    qApp->processEvents();
}

void *currentThread()
{
    return QThread::currentThread();
}

void *appThread()
{
    return QCoreApplication::instance()->thread();
}

QQmlEngine_ *newEngine(QObject_ *parent)
{
    return new QQmlEngine(reinterpret_cast<QObject *>(parent));
}

QQmlContext_ *engineRootContext(QQmlEngine_ *engine)
{
    return reinterpret_cast<QQmlEngine *>(engine)->rootContext();
}

void engineSetContextForObject(QQmlEngine_ *engine, QObject_ *object)
{
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    QObject *qobject = reinterpret_cast<QObject *>(object);

    QQmlEngine::setContextForObject(qobject, qengine->rootContext());
}

void engineSetOwnershipCPP(QQmlEngine_ *engine, QObject_ *object)
{
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    QObject *qobject = reinterpret_cast<QObject *>(object);

    qengine->setObjectOwnership(qobject, QQmlEngine::CppOwnership);
}

void engineSetOwnershipJS(QQmlEngine_ *engine, QObject_ *object)
{
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    QObject *qobject = reinterpret_cast<QObject *>(object);

    qengine->setObjectOwnership(qobject, QQmlEngine::JavaScriptOwnership);
}

QQmlComponent_ *newComponent(QQmlEngine_ *engine, QObject_ *parent)
{
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    //QObject *qparent = reinterpret_cast<QObject *>(parent);
    QQmlComponent *qcomponent = new QQmlComponent(qengine);
    // Qt 5.2.0 returns NULL on qmlEngine(qcomponent) without this.
    QQmlEngine::setContextForObject(qcomponent, qengine->rootContext());
    return qcomponent;
}

class GoImageProvider : public QQuickImageProvider {

    // TODO Destroy this when engine is destroyed.

    public:

    GoImageProvider(void *imageFunc) : QQuickImageProvider(QQmlImageProviderBase::Image), imageFunc(imageFunc) {};

    virtual QImage requestImage(const QString &id, QSize *size, const QSize &requestedSize)
    {
        QByteArray ba = id.toUtf8();
        int width = 0, height = 0;
        if (requestedSize.isValid()) {
            width = requestedSize.width();
            height = requestedSize.height();
        }
        QImage *ptr = reinterpret_cast<QImage *>(hookRequestImage(imageFunc, (char*)ba.constData(), ba.size(), width, height));
        QImage image = *ptr;
        delete ptr;

        *size = image.size();
        if (requestedSize.isValid() && requestedSize != *size) {
            image = image.scaled(requestedSize, Qt::KeepAspectRatio);
        }
        return image;
    };

    private:

    void *imageFunc;
};

void engineAddImageProvider(QQmlEngine_ *engine, QString_ *providerId, void *imageFunc)
{
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    QString *qproviderId = reinterpret_cast<QString *>(providerId);

    qengine->addImageProvider(*qproviderId, new GoImageProvider(imageFunc));
}

void componentLoadURL(QQmlComponent_ *component, const char *url, int urlLen)
{
    QByteArray qurl(url, urlLen);
    QString qsurl = QString::fromUtf8(qurl);
    reinterpret_cast<QQmlComponent *>(component)->loadUrl(qsurl);
}

void componentSetData(QQmlComponent_ *component, const char *data, int dataLen, const char *url, int urlLen)
{
    QByteArray qdata(data, dataLen);
    QByteArray qurl(url, urlLen);
    QString qsurl = QString::fromUtf8(qurl);
    reinterpret_cast<QQmlComponent *>(component)->setData(qdata, qsurl);
}

char *componentErrorString(QQmlComponent_ *component)
{
    QQmlComponent *qcomponent = reinterpret_cast<QQmlComponent *>(component);
    if (qcomponent->isReady()) {
        return NULL;
    }
    if (qcomponent->isError()) {
        QByteArray ba = qcomponent->errorString().toUtf8();
        return local_strdup(ba.constData());
    }
    return local_strdup("component is not ready (why!?)");
}

QObject_ *componentCreate(QQmlComponent_ *component, QQmlContext_ *context)
{
    QQmlComponent *qcomponent = reinterpret_cast<QQmlComponent *>(component);
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);

    if (!qcontext) {
        qcontext = qmlContext(qcomponent);
    }
    return qcomponent->create(qcontext);
}

QQuickWindow_ *componentCreateWindow(QQmlComponent_ *component, QQmlContext_ *context)
{
    QQmlComponent *qcomponent = reinterpret_cast<QQmlComponent *>(component);
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);

    if (!qcontext) {
        qcontext = qmlContext(qcomponent);
    }
    QObject *obj = qcomponent->create(qcontext);
    if (!objectIsWindow(obj)) {
        QQuickView *view = new QQuickView(qmlEngine(qcomponent), 0);
        view->setContent(qcomponent->url(), qcomponent, obj);
        view->setResizeMode(QQuickView::SizeRootObjectToView);
        obj = view;
    }
    return obj;
}

// Workaround for bug https://bugs.launchpad.net/bugs/1179716
struct DoShowWindow : public QQuickWindow {
    void show() {
        QQuickWindow::show();
        QResizeEvent resize(size(), size());
        resizeEvent(&resize);
    }
};

void windowShow(QQuickWindow_ *win)
{
    reinterpret_cast<DoShowWindow *>(win)->show();
}

void windowHide(QQuickWindow_ *win)
{
    reinterpret_cast<QQuickWindow *>(win)->hide();
}

uintptr_t windowPlatformId(QQuickWindow_ *win)
{
    return reinterpret_cast<QQuickWindow *>(win)->winId();
}

void windowConnectHidden(QQuickWindow_ *win)
{
    QQuickWindow *qwin = reinterpret_cast<QQuickWindow *>(win);
    QObject::connect(qwin, &QWindow::visibleChanged, [=](bool visible){
        if (!visible) {
            hookWindowHidden(win);
        }
    });
}

QObject_ *windowRootObject(QQuickWindow_ *win)
{
    if (objectIsView(win)) {
        return reinterpret_cast<QQuickView *>(win)->rootObject();
    }
    return win;
}

QImage_ *windowGrabWindow(QQuickWindow_ *win)
{
    QQuickWindow *qwin = reinterpret_cast<QQuickWindow *>(win);
    QImage *image = new QImage;
    *image = qwin->grabWindow().convertToFormat(QImage::Format_ARGB32_Premultiplied);
    return image;
}

QImage_ *newImage(int width, int height)
{
    return new QImage(width, height, QImage::Format_ARGB32_Premultiplied);
}

void delImage(QImage_ *image)
{
    delete reinterpret_cast<QImage *>(image);
}

void imageSize(QImage_ *image, int *width, int *height)
{
    QImage *qimage = reinterpret_cast<QImage *>(image);
    *width = qimage->width();
    *height = qimage->height();
}

unsigned char *imageBits(QImage_ *image)
{
    QImage *qimage = reinterpret_cast<QImage *>(image);
    return qimage->bits();
}

const unsigned char *imageConstBits(QImage_ *image)
{
    QImage *qimage = reinterpret_cast<QImage *>(image);
    return qimage->constBits();
}

void contextSetObject(QQmlContext_ *context, QObject_ *value)
{
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);
    QObject *qvalue = reinterpret_cast<QObject *>(value);

    // Give qvalue an engine reference if it doesn't yet have one.
    if (!qmlEngine(qvalue)) {
        QQmlEngine::setContextForObject(qvalue, qcontext->engine()->rootContext());
    }

    qcontext->setContextObject(qvalue);
}

void contextSetProperty(QQmlContext_ *context, QString_ *name, DataValue *value)
{
    const QString *qname = reinterpret_cast<QString *>(name);
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);

    QVariant var;
    unpackDataValue(value, &var);

    // Give qvalue an engine reference if it doesn't yet have one .
    QObject *obj = var.value<QObject *>();
    if (obj && !qmlEngine(obj)) {
        QQmlEngine::setContextForObject(obj, qcontext);
    }

    qcontext->setContextProperty(*qname, var);
}

void contextGetProperty(QQmlContext_ *context, QString_ *name, DataValue *result)
{
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);
    const QString *qname = reinterpret_cast<QString *>(name);

    QVariant var = qcontext->contextProperty(*qname);
    packDataValue(&var, result);
}

QQmlContext_ *contextSpawn(QQmlContext_ *context)
{
    QQmlContext *qcontext = reinterpret_cast<QQmlContext *>(context);
    return new QQmlContext(qcontext);
}

void delObject(QObject_ *object)
{
    delete reinterpret_cast<QObject *>(object);
}

void delObjectLater(QObject_ *object)
{
    reinterpret_cast<QObject *>(object)->deleteLater();
}

const char *objectTypeName(QObject_ *object)
{
    return reinterpret_cast<QObject *>(object)->metaObject()->className();
}

int objectGetProperty(QObject_ *object, const char *name, DataValue *result)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);
    
    QVariant var = qobject->property(name);
    packDataValue(&var, result);

    if (!var.isValid() && qobject->metaObject()->indexOfProperty(name) == -1) {
            // TODO May have to check the dynamic property names too.
            return 0;
    }
    return 1;
}

error *objectSetProperty(QObject_ *object, const char *name, DataValue *value)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);
    QVariant var;
    unpackDataValue(value, &var);

    // Give qvalue an engine reference if it doesn't yet have one.
    QObject *obj = var.value<QObject *>();
    if (obj && !qmlEngine(obj)) {
        QQmlContext *context = qmlContext(qobject);
        if (context) {
            QQmlEngine::setContextForObject(obj, context);
        }
    }

    // Check that the types are compatible. There's probably more to be done here.
    const QMetaObject *metaObject = qobject->metaObject();
    int propIndex = metaObject->indexOfProperty(name);
    if (propIndex == -1) {
            return errorf("cannot set non-existent property \"%s\" on type %s", name, qobject->metaObject()->className());
    }

    QMetaProperty prop = metaObject->property(propIndex);
    int propType = prop.userType();
    void *valueArg;
    if (propType == QMetaType::QVariant) {
        valueArg = (void *)&var;
    } else {
        int varType = var.userType();
        QVariant saved = var;
        if (propType != varType && !var.convert(propType)) {
            if (varType == QMetaType::QObjectStar) {
                return errorf("cannot set property \"%s\" with type %s to value of %s*",
                        name, QMetaType::typeName(propType), saved.value<QObject*>()->metaObject()->className());
            } else {
                return errorf("cannot set property \"%s\" with type %s to value of %s",
                        name, QMetaType::typeName(propType), QMetaType::typeName(varType));
            }
        }
        valueArg = (void *)var.constData();
    }

    int status = -1;
    int flags = 0;
    void *args[] = {valueArg, 0, &status, &flags};
    QMetaObject::metacall(qobject, QMetaObject::WriteProperty, propIndex, args);
    return 0;
}

error *objectInvoke(QObject_ *object, const char *method, int methodLen, DataValue *resultdv, DataValue *paramsdv, int paramsLen)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);

    QVariant result;
    QVariant param[MaxParams];
    QGenericArgument arg[MaxParams];
    for (int i = 0; i < paramsLen; i++) {
        unpackDataValue(&paramsdv[i], &param[i]);
        arg[i] = Q_ARG(QVariant, param[i]);
    }
    if (paramsLen > 10) {
        panicf("fix the parameter dispatching");
    }

    const QMetaObject *metaObject = qobject->metaObject();
    // Walk backwards so descendants have priority.
    for (int i = metaObject->methodCount()-1; i >= 0; i--) {
        QMetaMethod metaMethod = metaObject->method(i);
        QMetaMethod::MethodType methodType = metaMethod.methodType();
        if (methodType == QMetaMethod::Method || methodType == QMetaMethod::Slot) {
            QByteArray name = metaMethod.name();
            if (name.length() == methodLen && qstrncmp(name.constData(), method, methodLen) == 0) {
                if (metaMethod.parameterCount() < paramsLen) {
                    // TODO Might continue looking to see if a different signal has the same name and enough arguments.
                    return errorf("method \"%s\" has too few parameters for provided arguments", method);
                }

                bool ok;
                if (metaMethod.returnType() == QMetaType::Void) {
                    ok = metaMethod.invoke(qobject, Qt::DirectConnection, 
                        arg[0], arg[1], arg[2], arg[3], arg[4], arg[5], arg[6], arg[7], arg[8], arg[9]);
                } else {
                    ok = metaMethod.invoke(qobject, Qt::DirectConnection, Q_RETURN_ARG(QVariant, result),
                        arg[0], arg[1], arg[2], arg[3], arg[4], arg[5], arg[6], arg[7], arg[8], arg[9]);
                }
                if (!ok) {
                    return errorf("invalid parameters to method \"%s\"", method);
                }

                packDataValue(&result, resultdv);
                return 0;
            }
        }
    }

    return errorf("object does not expose a method \"%s\"", method);
}

void objectFindChild(QObject_ *object, QString_ *name, DataValue *resultdv)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);
    QString *qname = reinterpret_cast<QString *>(name);
    
    QVariant var;
    QObject *result = qobject->findChild<QObject *>(*qname);
    if (result) {
        var.setValue(result);
    }
    packDataValue(&var, resultdv);
}

void objectSetParent(QObject_ *object, QObject_ *parent)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);
    QObject *qparent = reinterpret_cast<QObject *>(parent);

    qobject->setParent(qparent);
}

error *objectConnect(QObject_ *object, const char *signal, int signalLen, QQmlEngine_ *engine, void *func, int argsLen)
{
    QObject *qobject = reinterpret_cast<QObject *>(object);
    QQmlEngine *qengine = reinterpret_cast<QQmlEngine *>(engine);
    QByteArray qsignal(signal, signalLen);

    const QMetaObject *meta = qobject->metaObject();
    // Walk backwards so descendants have priority.
    for (int i = meta->methodCount()-1; i >= 0; i--) {
            QMetaMethod method = meta->method(i);
            if (method.methodType() == QMetaMethod::Signal) {
                QByteArray name = method.name();
                if (name.length() == signalLen && qstrncmp(name.constData(), signal, signalLen) == 0) {
                    if (method.parameterCount() < argsLen) {
                        // TODO Might continue looking to see if a different signal has the same name and enough arguments.
                        return errorf("signal \"%s\" has too few parameters for provided function", name.constData());
                    }
                    Connector *connector = new Connector(qobject, method, qengine, func, argsLen);
                    const QMetaObject *connmeta = connector->metaObject();
                    QObject::connect(qobject, method, connector, connmeta->method(connmeta->methodOffset()));
                    return 0;
                }
            }
    }
    // Cannot use constData here as the byte array is not null-terminated.
    return errorf("object does not expose a \"%s\" signal", qsignal.data());
}

QQmlContext_ *objectContext(QObject_ *object)
{
    return qmlContext(static_cast<QObject *>(object));
}

int objectIsComponent(QObject_ *object)
{
    QObject *qobject = static_cast<QObject *>(object);
    return dynamic_cast<QQmlComponent *>(qobject) ? 1 : 0;
}

int objectIsWindow(QObject_ *object)
{
    QObject *qobject = static_cast<QObject *>(object);
    return dynamic_cast<QQuickWindow *>(qobject) ? 1 : 0;
}

int objectIsView(QObject_ *object)
{
    QObject *qobject = static_cast<QObject *>(object);
    return dynamic_cast<QQuickView *>(qobject) ? 1 : 0;
}

error *objectGoAddr(QObject_ *object, GoAddr **addr)
{
    QObject *qobject = static_cast<QObject *>(object);
    GoValue *goValue = dynamic_cast<GoValue *>(qobject);
    if (goValue) {
        *addr = goValue->addr;
        return 0;
    }
    GoPaintedValue *goPaintedValue = dynamic_cast<GoPaintedValue *>(qobject);
    if (goPaintedValue) {
        *addr = goPaintedValue->addr;
        return 0;
    }
    return errorf("QML object is not backed by a Go value");
}

QString_ *newString(const char *data, int len)
{
    // This will copy data only once.
    QByteArray ba = QByteArray::fromRawData(data, len);
    return new QString(ba);
}

void delString(QString_ *s)
{
    delete reinterpret_cast<QString *>(s);
}

GoValue_ *newGoValue(GoAddr *addr, GoTypeInfo *typeInfo, QObject_ *parent)
{
    QObject *qparent = reinterpret_cast<QObject *>(parent);
    if (typeInfo->paint) {
        return new GoPaintedValue(addr, typeInfo, qparent);
    }
    return new GoValue(addr, typeInfo, qparent);
}

void goValueActivate(GoValue_ *value, GoTypeInfo *typeInfo, int addrOffset)
{
    GoMemberInfo *fieldInfo = typeInfo->fields;
    for (int i = 0; i < typeInfo->fieldsLen; i++) {
        if (fieldInfo->addrOffset == addrOffset) {
            if (typeInfo->paint) {
                static_cast<GoPaintedValue *>(value)->activate(fieldInfo->metaIndex);
            } else {
                static_cast<GoValue *>(value)->activate(fieldInfo->metaIndex);
            }
            return;
        }
        fieldInfo++;
    }

    // TODO Return an error; probably an unexported field.
}

void unpackDataValue(DataValue *value, QVariant_ *var)
{
    QVariant *qvar = reinterpret_cast<QVariant *>(var);
    switch (value->dataType) {
    case DTString:
        *qvar = QString::fromUtf8(*(char **)value->data, value->len);
        break;
    case DTBool:
        *qvar = bool(*(char *)(value->data) != 0);
        break;
    case DTInt64:
        *qvar = *(qint64*)(value->data);
        break;
    case DTInt32:
        *qvar = *(qint32*)(value->data);
        break;
    case DTUint64:
        *qvar = *(quint64*)(value->data);
        break;
    case DTUint32:
        *qvar = *(quint32*)(value->data);
        break;
    case DTFloat64:
        *qvar = *(double*)(value->data);
        break;
    case DTFloat32:
        *qvar = *(float*)(value->data);
        break;
    case DTColor:
        *qvar = QColor::fromRgba(*(QRgb*)(value->data));
        break;
    case DTVariantList:
        *qvar = **(QVariantList**)(value->data);
        delete *(QVariantList**)(value->data);
        break;
    case DTObject:
        qvar->setValue(*(QObject**)(value->data));
        break;
    case DTInvalid:
        // null would be more natural, but an invalid variant means
        // it has proper semantics when dealing with non-qml qt code.
        //qvar->setValue(QJSValue(QJSValue::NullValue));
        qvar->clear();
        break;
    default:
        panicf("unknown data type: %d", value->dataType);
        break;
    }
}

void packDataValue(QVariant_ *var, DataValue *value)
{
    QVariant *qvar = reinterpret_cast<QVariant *>(var);

    // Some assumptions are made below regarding the size of types.
    // There's apparently no better way to handle this since that's
    // how the types with well defined sizes (qint64) are mapped to
    // meta-types (QMetaType::LongLong).
    switch ((int)qvar->type()) {
    case QVariant::Invalid:
        value->dataType = DTInvalid;
        break;
    case QMetaType::QUrl:
        *qvar = qvar->value<QUrl>().toString();
        // fallthrough
    case QMetaType::QString:
        {
            value->dataType = DTString;
            QByteArray ba = qvar->toByteArray();
            *(char**)(value->data) = local_strdup(ba.constData());
            value->len = ba.size();
            break;
        }
    case QMetaType::Bool:
        value->dataType = DTBool;
        *(qint8*)(value->data) = (qint8)qvar->toInt();
        break;
    case QMetaType::LongLong:
        // Some of these entries will have to be fixed when handling platforms
        // where sizeof(long long) != 8 or sizeof(int) != 4.
        value->dataType = DTInt64;
        *(qint64*)(value->data) = qvar->toLongLong();
        break;
    case QMetaType::ULongLong:
        value->dataType = DTUint64;
        *(quint64*)(value->data) = qvar->toLongLong();
        break;
    case QMetaType::Int:
        value->dataType = DTInt32;
        *(qint32*)(value->data) = qvar->toInt();
        break;
    case QMetaType::UInt:
        value->dataType = DTUint32;
        *(quint32*)(value->data) = qvar->toUInt();
        break;
    case QMetaType::VoidStar:
        value->dataType = DTUintptr;
        *(uintptr_t*)(value->data) = (uintptr_t)qvar->value<void *>();
        break;
    case QMetaType::Double:
        value->dataType = DTFloat64;
        *(double*)(value->data) = qvar->toDouble();
        break;
    case QMetaType::Float:
        value->dataType = DTFloat32;
        *(float*)(value->data) = qvar->toFloat();
        break;
    case QMetaType::QColor:
        value->dataType = DTColor;
        *(unsigned int*)(value->data) = qvar->value<QColor>().rgba();
        break;
    case QMetaType::QVariantList:
        {
            QVariantList varlist = qvar->toList();
            int len = varlist.size();
            DataValue *dvlist = (DataValue *) malloc(sizeof(DataValue) * len);
            for (int i = 0; i < len; i++) {
                packDataValue((void*)&varlist.at(i), &dvlist[i]);
            }
            value->dataType = DTValueList;
            value->len = len;
            *(DataValue**)(value->data) = dvlist;
        }
        break;
    case QMetaType::QVariantMap:
        {
            QVariantMap varmap = qvar->toMap();
            int len = varmap.size() * 2;
            DataValue *dvlist = (DataValue *) malloc(sizeof(DataValue) * len);
            QMapIterator<QString, QVariant> it(varmap);
            for (int i = 0; i < len; i += 2) {
                if (!it.hasNext()) {
                    panicf("QVariantMap mutated during iteration");
                }
                it.next();
                QVariant key = it.key();
                QVariant val = it.value();
                packDataValue((void*)&key, &dvlist[i]);
                packDataValue((void*)&val, &dvlist[i+1]);
            }
            value->dataType = DTValueMap;
            value->len = len;
            *(DataValue**)(value->data) = dvlist;
        }
        break;
    case QMetaType::User:
        {
            static const int qjstype = QVariant::fromValue(QJSValue()).userType();
            if (qvar->userType() == qjstype) {
                auto var = qvar->value<QJSValue>().toVariant();
                packDataValue(&var, value);
            }
        }
        break;
    default:
        if (qvar->type() == (int)QMetaType::QObjectStar || qvar->canConvert<QObject *>()) {
            QObject *qobject = qvar->value<QObject *>();
            GoValue *goValue = dynamic_cast<GoValue *>(qobject);
            if (goValue) {
                value->dataType = DTGoAddr;
                *(void **)(value->data) = goValue->addr;
                break;
            }
            GoPaintedValue *goPaintedValue = dynamic_cast<GoPaintedValue *>(qobject);
            if (goPaintedValue) {
                value->dataType = DTGoAddr;
                *(void **)(value->data) = goPaintedValue->addr;
                break;
            }
            value->dataType = DTObject;
            *(void **)(value->data) = qobject;
            break;
        }
        {
            QQmlListReference ref = qvar->value<QQmlListReference>();
            if (ref.isValid() && ref.canCount() && ref.canAt()) {
                int len = ref.count();
                DataValue *dvlist = (DataValue *) malloc(sizeof(DataValue) * len);
                QVariant elem;
                for (int i = 0; i < len; i++) {
                    elem.setValue(ref.at(i));
                    packDataValue(&elem, &dvlist[i]);
                }
                value->dataType = DTValueList;
                value->len = len;
                *(DataValue**)(value->data) = dvlist;
                break;
            }
        }
        if (qstrncmp(qvar->typeName(), "QQmlListProperty<", 17) == 0) {
            QQmlListProperty<QObject> *list = reinterpret_cast<QQmlListProperty<QObject>*>(qvar->data());
            if (list->count && list->at) {
                int len = list->count(list);
                DataValue *dvlist = (DataValue *) malloc(sizeof(DataValue) * len);
                QVariant elem;
                for (int i = 0; i < len; i++) {
                    elem.setValue(list->at(list, i));
                    packDataValue(&elem, &dvlist[i]);
                }
                value->dataType = DTValueList;
                value->len = len;
                *(DataValue**)(value->data) = dvlist;
                break;
            }
        }
        panicf("unsupported variant type: %d (%s)", qvar->type(), qvar->typeName());
        break;
    }
}

QVariantList_ *newVariantList(DataValue *list, int len)
{
    QVariantList *vlist = new QVariantList();
    vlist->reserve(len);
    for (int i = 0; i < len; i++) {
        QVariant var;
        unpackDataValue(&list[i], &var);
        vlist->append(var);
    }
    return vlist;
}

QObject *listPropertyAt(QQmlListProperty<QObject> *list, int i)
{
    return reinterpret_cast<QObject *>(hookListPropertyAt(list->data, (intptr_t)list->dummy1, (intptr_t)list->dummy2, i));
}

int listPropertyCount(QQmlListProperty<QObject> *list)
{
    return hookListPropertyCount(list->data, (intptr_t)list->dummy1, (intptr_t)list->dummy2);
}

void listPropertyAppend(QQmlListProperty<QObject> *list, QObject *obj)
{
    hookListPropertyAppend(list->data, (intptr_t)list->dummy1, (intptr_t)list->dummy2, obj);
}

void listPropertyClear(QQmlListProperty<QObject> *list)
{
    hookListPropertyClear(list->data, (intptr_t)list->dummy1, (intptr_t)list->dummy2);
}

QQmlListProperty_ *newListProperty(GoAddr *addr, intptr_t reflectIndex, intptr_t setIndex)
{
    QQmlListProperty<QObject> *list = new QQmlListProperty<QObject>();
    list->data = addr;
    list->dummy1 = (void*)reflectIndex;
    list->dummy2 = (void*)setIndex;
    list->at = listPropertyAt;
    list->count = listPropertyCount;
    list->append = listPropertyAppend;
    list->clear = listPropertyClear;
    return list;
}

void internalLogHandler(QtMsgType severity, const QMessageLogContext &context, const QString &text)
{
    if (context.file == NULL) return;

    QByteArray textba = text.toUtf8();
    LogMessage message = {severity, textba.constData(), textba.size(), context.file, (int)strlen(context.file), context.line};
    hookLogHandler(&message);
}

void installLogHandler()
{
    qInstallMessageHandler(internalLogHandler);
}


extern bool qRegisterResourceData(int version, const unsigned char *tree, const unsigned char *name, const unsigned char *data);
extern bool qUnregisterResourceData(int version, const unsigned char *tree, const unsigned char *name, const unsigned char *data);

void registerResourceData(int version, char *tree, char *name, char *data)
{
    qRegisterResourceData(version, (unsigned char*)tree, (unsigned char*)name, (unsigned char*)data);
}

void unregisterResourceData(int version, char *tree, char *name, char *data)
{
    qUnregisterResourceData(version, (unsigned char*)tree, (unsigned char*)name, (unsigned char*)data);
}

// vim:ts=4:sw=4:et:ft=cpp
