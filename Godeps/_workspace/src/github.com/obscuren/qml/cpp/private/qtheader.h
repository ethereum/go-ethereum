#ifndef QTPRIVATE_H
#define QTPRIVATE_H

#include <QtCore/qglobal.h>

#define QT_MAJOR_ (QT_VERSION>>16)
#define QT_MINOR_ (QT_VERSION>>8&0xFF)
#define QT_MICRO_ (QT_VERSION&0xFF)

#if QT_MAJOR_ == 5
#define QT_MAJOR 5
#else 
#error Unupported Qt major version. Please report.
#endif

#if QT_MINOR_ == 0
#define QT_MINOR 0
#elif QT_MINOR_ == 1
#define QT_MINOR 1
#elif QT_MINOR_ == 2
#define QT_MINOR 2
#elif QT_MINOR_ == 3
#define QT_MINOR 3
#elif QT_MINOR_ == 4
#define QT_MINOR 4
#elif QT_MINOR_ == 5
#define QT_MINOR 5
#elif QT_MINOR_ == 6
#define QT_MINOR 6
#elif QT_MINOR_ == 7
#define QT_MINOR 7
#elif QT_MINOR_ == 8
#define QT_MINOR 8
#elif QT_MINOR_ == 9
#define QT_MINOR 9
#elif QT_MINOR_ == 10
#define QT_MINOR 10
#else 
#error Unupported Qt minor version. Please report.
#endif

#if QT_MICRO_ == 0
#define QT_MICRO 0
#elif QT_MICRO_ == 1
#define QT_MICRO 1
#elif QT_MICRO_ == 2
#define QT_MICRO 2
#elif QT_MICRO_ == 3
#define QT_MICRO 3
#elif QT_MICRO_ == 4
#define QT_MICRO 4
#elif QT_MICRO_ == 5
#define QT_MICRO 5
#elif QT_MICRO_ == 6
#define QT_MICRO 6
#elif QT_MICRO_ == 7
#define QT_MICRO 7
#elif QT_MICRO_ == 8
#define QT_MICRO 8
#elif QT_MICRO_ == 9
#define QT_MICRO 9
#elif QT_MICRO_ == 10
#define QT_MICRO 10
#else 
#error Unupported Qt micro version. Please report.
#endif

#define QT_PRIVATE_HEADER(dir,file) <dir/QT_MAJOR.QT_MINOR.QT_MICRO/dir/private/file>

#endif // QTPRIVATE_H
