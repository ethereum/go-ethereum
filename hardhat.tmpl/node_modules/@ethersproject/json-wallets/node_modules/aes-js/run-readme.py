import re

reJavaScript = re.compile('```javascript((.|\n)*?)```')
readmeData = file('README.md').read()

print 'const aesjs = require("./index.js");'
for (example, nl) in reJavaScript.findall(readmeData):
    print 'console.log("=====================");'
    print '(function() {'
    print '    try {'
    print 'console.log(%r)' % example
    for line in example.split('\n'):
        print (' ' * 8) + line
    print '    } catch (error) { console.log("ERROR: ",  error); }'
    print '})();'
