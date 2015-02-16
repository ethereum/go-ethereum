/****************************************************************************
** Meta object code from reading C++ file 'testtype.h'
**
** Created by: The Qt Meta Object Compiler version 67 (Qt 5.2.1)
**
** WARNING! All changes made in this file will be lost!
*****************************************************************************/

#include "testtype.h"
#include <QtCore/qbytearray.h>
#include <QtCore/qmetatype.h>
#if !defined(Q_MOC_OUTPUT_REVISION)
#error "The header file 'testtype.h' doesn't include <QObject>."
#elif Q_MOC_OUTPUT_REVISION != 67
#error "This file was generated using the moc from 5.2.1. It"
#error "cannot be used with the include files from this version of Qt."
#error "(The moc has changed too much.)"
#endif

QT_BEGIN_MOC_NAMESPACE
struct qt_meta_stringdata_TestType_t {
    QByteArrayData data[10];
    char stringdata[119];
};
#define QT_MOC_LITERAL(idx, ofs, len) \
    Q_STATIC_BYTE_ARRAY_DATA_HEADER_INITIALIZER_WITH_OFFSET(len, \
    offsetof(qt_meta_stringdata_TestType_t, stringdata) + ofs \
        - idx * sizeof(QByteArrayData) \
    )
static const qt_meta_stringdata_TestType_t qt_meta_stringdata_TestType = {
    {
QT_MOC_LITERAL(0, 0, 8),
QT_MOC_LITERAL(1, 9, 15),
QT_MOC_LITERAL(2, 25, 0),
QT_MOC_LITERAL(3, 26, 13),
QT_MOC_LITERAL(4, 40, 5),
QT_MOC_LITERAL(5, 46, 15),
QT_MOC_LITERAL(6, 62, 15),
QT_MOC_LITERAL(7, 78, 20),
QT_MOC_LITERAL(8, 99, 9),
QT_MOC_LITERAL(9, 109, 8)
    },
    "TestType\0plainEmittedCpy\0\0PlainTestType\0"
    "plain\0plainEmittedRef\0plainEmittedPtr\0"
    "const PlainTestType*\0emitPlain\0voidAddr\0"
};
#undef QT_MOC_LITERAL

static const uint qt_meta_data_TestType[] = {

 // content:
       7,       // revision
       0,       // classname
       0,    0, // classinfo
       4,   14, // methods
       1,   44, // properties
       0,    0, // enums/sets
       0,    0, // constructors
       0,       // flags
       3,       // signalCount

 // signals: name, argc, parameters, tag, flags
       1,    1,   34,    2, 0x06,
       5,    1,   37,    2, 0x06,
       6,    1,   40,    2, 0x06,

 // methods: name, argc, parameters, tag, flags
       8,    0,   43,    2, 0x02,

 // signals: parameters
    QMetaType::Void, 0x80000000 | 3,    4,
    QMetaType::Void, 0x80000000 | 3,    4,
    QMetaType::Void, 0x80000000 | 7,    4,

 // methods: parameters
    QMetaType::Void,

 // properties: name, type, flags
       9, QMetaType::VoidStar, 0x00095001,

       0        // eod
};

void TestType::qt_static_metacall(QObject *_o, QMetaObject::Call _c, int _id, void **_a)
{
    if (_c == QMetaObject::InvokeMetaMethod) {
        TestType *_t = static_cast<TestType *>(_o);
        switch (_id) {
        case 0: _t->plainEmittedCpy((*reinterpret_cast< const PlainTestType(*)>(_a[1]))); break;
        case 1: _t->plainEmittedRef((*reinterpret_cast< const PlainTestType(*)>(_a[1]))); break;
        case 2: _t->plainEmittedPtr((*reinterpret_cast< const PlainTestType*(*)>(_a[1]))); break;
        case 3: _t->emitPlain(); break;
        default: ;
        }
    } else if (_c == QMetaObject::IndexOfMethod) {
        int *result = reinterpret_cast<int *>(_a[0]);
        void **func = reinterpret_cast<void **>(_a[1]);
        {
            typedef void (TestType::*_t)(const PlainTestType );
            if (*reinterpret_cast<_t *>(func) == static_cast<_t>(&TestType::plainEmittedCpy)) {
                *result = 0;
            }
        }
        {
            typedef void (TestType::*_t)(const PlainTestType & );
            if (*reinterpret_cast<_t *>(func) == static_cast<_t>(&TestType::plainEmittedRef)) {
                *result = 1;
            }
        }
        {
            typedef void (TestType::*_t)(const PlainTestType * );
            if (*reinterpret_cast<_t *>(func) == static_cast<_t>(&TestType::plainEmittedPtr)) {
                *result = 2;
            }
        }
    }
}

const QMetaObject TestType::staticMetaObject = {
    { &QObject::staticMetaObject, qt_meta_stringdata_TestType.data,
      qt_meta_data_TestType,  qt_static_metacall, 0, 0}
};


const QMetaObject *TestType::metaObject() const
{
    return QObject::d_ptr->metaObject ? QObject::d_ptr->dynamicMetaObject() : &staticMetaObject;
}

void *TestType::qt_metacast(const char *_clname)
{
    if (!_clname) return 0;
    if (!strcmp(_clname, qt_meta_stringdata_TestType.stringdata))
        return static_cast<void*>(const_cast< TestType*>(this));
    return QObject::qt_metacast(_clname);
}

int TestType::qt_metacall(QMetaObject::Call _c, int _id, void **_a)
{
    _id = QObject::qt_metacall(_c, _id, _a);
    if (_id < 0)
        return _id;
    if (_c == QMetaObject::InvokeMetaMethod) {
        if (_id < 4)
            qt_static_metacall(this, _c, _id, _a);
        _id -= 4;
    } else if (_c == QMetaObject::RegisterMethodArgumentMetaType) {
        if (_id < 4)
            *reinterpret_cast<int*>(_a[0]) = -1;
        _id -= 4;
    }
#ifndef QT_NO_PROPERTIES
      else if (_c == QMetaObject::ReadProperty) {
        void *_v = _a[0];
        switch (_id) {
        case 0: *reinterpret_cast< void**>(_v) = getVoidAddr(); break;
        }
        _id -= 1;
    } else if (_c == QMetaObject::WriteProperty) {
        _id -= 1;
    } else if (_c == QMetaObject::ResetProperty) {
        _id -= 1;
    } else if (_c == QMetaObject::QueryPropertyDesignable) {
        _id -= 1;
    } else if (_c == QMetaObject::QueryPropertyScriptable) {
        _id -= 1;
    } else if (_c == QMetaObject::QueryPropertyStored) {
        _id -= 1;
    } else if (_c == QMetaObject::QueryPropertyEditable) {
        _id -= 1;
    } else if (_c == QMetaObject::QueryPropertyUser) {
        _id -= 1;
    } else if (_c == QMetaObject::RegisterPropertyMetaType) {
        if (_id < 1)
            *reinterpret_cast<int*>(_a[0]) = -1;
        _id -= 1;
    }
#endif // QT_NO_PROPERTIES
    return _id;
}

// SIGNAL 0
void TestType::plainEmittedCpy(const PlainTestType _t1)
{
    void *_a[] = { 0, const_cast<void*>(reinterpret_cast<const void*>(&_t1)) };
    QMetaObject::activate(this, &staticMetaObject, 0, _a);
}

// SIGNAL 1
void TestType::plainEmittedRef(const PlainTestType & _t1)
{
    void *_a[] = { 0, const_cast<void*>(reinterpret_cast<const void*>(&_t1)) };
    QMetaObject::activate(this, &staticMetaObject, 1, _a);
}

// SIGNAL 2
void TestType::plainEmittedPtr(const PlainTestType * _t1)
{
    void *_a[] = { 0, const_cast<void*>(reinterpret_cast<const void*>(&_t1)) };
    QMetaObject::activate(this, &staticMetaObject, 2, _a);
}
QT_END_MOC_NAMESPACE
