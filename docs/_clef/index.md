---
title: Clef
root: ..
permalink: /clef/
---
{% for p in site.clef %}
* [{{ p.title }}]({% include link.html url=p.url %})
{% endfor %}