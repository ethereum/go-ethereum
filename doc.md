---
title: Other documents
root: ..
permalink: /doc/
---
{% for p in site.doc %}
* [{{ p.title }}]({% include link.html url=p.url %})
{% endfor %}
