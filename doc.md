---
title: Other documents
root: ..
---
{% for p in site.doc %}
* [{{ p.title }}]({% include link.html url=p.url %})
{% endfor %}