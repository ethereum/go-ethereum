#include "clipboard.h"

#include <QClipboard>

Clipboard::Clipboard()
{
	connect(QApplication::clipboard(), &QClipboard::dataChanged, [this] { emit clipboardChanged();});
}

QString Clipboard::get() const
{
	QClipboard *clipboard = QApplication::clipboard();
	return clipboard->text();
}

void Clipboard::toClipboard(QString _text)
{
	QClipboard *clipboard = QApplicationion::clipboard();
	clipboard->setText(_text);
}
