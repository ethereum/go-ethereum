#pragma once

#ifdef __cplusplus
extern "C" {
#endif

class Clipboard : public QObject
{
	Q_OBJECT
	Q_PROPERTY(QString get READ get WRITE toClipboard NOTIFY clipboardChanged)
public:
	Clipboard();
	virtual ~Clipboard(){}

	Q_INVOKABLE void toClipboard(QString _text);

signals:
	void clipboardChanged();
};

#ifdef __cplusplus
} // extern "C"
#endif
