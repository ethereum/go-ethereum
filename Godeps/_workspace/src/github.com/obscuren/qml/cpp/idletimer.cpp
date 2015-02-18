#include <QBasicTimer>
#include <QThread>
#include <QDebug>
#include <mutex>

#include "capi.h"

class IdleTimer : public QObject
{
    Q_OBJECT

    public:

    static IdleTimer *singleton() {
        static IdleTimer singleton;
        return &singleton;
    }

    void init(int32_t *guiIdleRun)
    {
        this->guiIdleRun = guiIdleRun;
    }

    Q_INVOKABLE void start()
    {
        timer.start(0, this);
    }

    protected:

    void timerEvent(QTimerEvent *event)
    {
        __sync_synchronize();
        if (*guiIdleRun > 0) {
            hookIdleTimer();
        } else {
            timer.stop();
        }
    }

    private:

    int32_t *guiIdleRun;

    QBasicTimer timer;    
};

void idleTimerInit(int32_t *guiIdleRun)
{
    IdleTimer::singleton()->init(guiIdleRun);
}

void idleTimerStart()
{
    QMetaObject::invokeMethod(IdleTimer::singleton(), "start", Qt::QueuedConnection);
}

// vim:ts=4:sw=4:et:ft=cpp
