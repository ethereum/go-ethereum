# URLs in DAPP browsers 

URLs should contain all allowable urls in browsers and _all_ `http(s)` urls that resolve in a usual browser must resolve the same way. 

All urls not conforming to the existing urls scheme must still resemble the current urls scheme.

```
<protocol>://<source>/<path>
```

Irrespective of the main protocol, `<source>` should be resolved with our version of DNS (`NameReg` (ename registration contract on ethereum) and/or via swarm signed version stream. 

In the special case of the bzz protocol, `<source>` must resolve to a Swarm hash of the content (in other words, the root key of the content). This content is assumed to be of mime type `application/bzz-sitemap+json` the only mime-type directly handled by Swarm. 

# Swarm manifests

A Swarm manifest is a json formatted description of url routing. 
The swarm manifest allows swarm documents to act as file systems or webservers. 
Their mime type is `application/bzz-sitemap+json`
Manifest has the following attributes:

- `entries`: an array of route configurations
- `host`: eth host name registered (or to register) with NameReg
- `number`: position index (increasing integers) of manifest within channel, 
- `auth`: devp2p cryptohandshake public key(s), signed number 
- `first`: root key of initial state of the stream 
- `previous`: previous state of stream 

A route descriptor manifest entry json object has the following attributes:

- `path`: a path relative to the url that resolved to the manifest (_optional, with empty default_)
- `hash`: key of the content to be looked up by swarm (_optional_)
- `link`: relative path or external link (_optional_)
- `contentType`: mime type of the content (_optional, `application/bzz-server` by default_)
- `status`: optional http status code to pass back to the server (_optional, 200 by default_)
- `cache`: cache entry, etag? and other header options (_optional_)
- `www`: alternative old web address that the route replicates: e.g., `http://eth:bzz@google.com` (_optional_)

If `path` is an empty string or is missing, the path matches the _document-root_ of the DAPP.
If `contentType` is empty or missing, manifest if assumed by default.

(NOTE: Unclear. When no path matches and there is no fallback path e.g. a root `/` path with hash specified, it should return a simple 404 status code)

# Url resolution

Given 

```
 bzz://<source>/<path>
```

in the browser, the following steps need to happen:

- the browser sees that its bzz protocol `<source>/<path>` is passed to the *bzz protocol handler*,
- the handler checks if `<source>` is a hash. If not it resolves to a hash via NameReg and signed version table, see below
- the bzz protocol handler first retrieves the content for the hash (with integrity check) which it interprets as a manifest file (`application/bzz-sitemap+json`),
- this manifest file is then parsed, read and the json array element with the longest prefix `p` of `<path>` is looked up. I.e., `p` is the longest prefix such that `<path> == p'/p''`. (If the longest prefix is 0 length, the row with `<path> == ""` (or left out) is chosen.)
- as a special case, trailing forward slashes are ignored so all variants will match the directory,
- the protocol then looks up content for `p'` and serves it to the browser together with the status code and content type. 
- if content is of type manifest, bzz retrieves it and repeats the steps using `p''` to match the manifest's `<path>` values against,
- the url relative path is set to `p''` 
- if the url looked up is an old-world http site, then a standard http client call is sufficient.

### Example 1

```js
{
   entries: [
     {
        "path": "cv.pdf",
        "contentType": "document/pdf",
        "hash": "sdfhsd76ftsd86ft76sdgf78h7tg", 
      }
   ]
}
```

where the hash is the hash of the actual file `cv.pdf`.

If this manifest hashes to `dafghjfgsdgfjfgsdjfgsd`, then `bzz://dafghjfgsdgfjfgsdjfgsd/cv.pdf` will serve `cv.pdf`

Now you can register the manifest hash with NameReg to resolve `my-website` the file as follows: 

```
   http://my-website/cv.pdf 
```

serves `cv.pdf`

### Example 2
Imagine you have a DAPP called _chat_ and host it under  
your local directory `<dir>` looks like this:

```
  index.html
  img/logo.gif
  img/avatars/fefe.jpg
  img/avatars/index.html
```

the webserver has the following routing rules:

```
  -> <dir>/index.html 
  <unkwown> -> <dir>/index.html # where <unknown> != index.html
  img/logo.gif -> <dir>/img/logo.gif 
  img/avatars -> <dir>img/avatars/index.html
  img/avatars/fefe.jpg -> <dir>/img/avatars/fefe.jpg
  img/avatars/<unknown>.jpg <dir>/img/avatars/index.html # where <unknown> != fefe.jpg
```

Now you can alternatively host your app in Swarm by creating the following manifest:

```js
{ 
  "entries": [
  { "hash": HASH(<dir>/index.html) },
  { "path": "index.html", "hash": HASH(<dir>/index.html) },
  { "path": "img/logo.gif", "hash": HASH(<dir>/img/logo.gif) },
  { "path": "img/avatars/", "hash": HASH(<dir>/img/avatars/index.html) },
  { "path": "img/avatars/fefe.jpg", "hash": HASH(img/avatars/fefe.jpg) }
  ]
}
```

# Swarm webservers

Swarm webservers are simply bzz site manifest files routing relative paths to static assets.
Manifest route entries specify metadata: http header values, etag, redirects, links, etc.  

In a typical scenario, the developer has a website within a working copy directory on their dev environment and they want to create a decentralised version of their site.

They then register the host domain with ethereum NameReg or swarm signed version stream, upload all desired static assets to swarm, and produce a site manifest.

In order to facilitate the creation of the manifest file for existing web projects,  a native API and a command line utility are provided to automatically generate manifest files from a directory.

## ArcHive API 

A native API and a command line utility are provided to automatically swarmify document collections. 
constructor parameters:

- `template`: manifest template: the entries found in the directory scan are merged into this template to yield the resulting site-map. Note that this template can be considered a config file to the archiver.

The archiver can be called multiple times scanning multiple directories.

runtime parameters:
- `path`: path to directory relative routes in the template matched against directory paths under `path` (_optional_, '.' by default).
- `not-found`: errorchange to be used when asset is not found: for 404, (_optional_, `index.html`)
- `register-names` use eth NameReg to register public key and this version is pushed to swarm mutable store (_optional_, _false_)
- `without-scan` only consider paths given in template (_optional_, by default _false_: in template, scan directory and add/merge all readable content to manifest)
- `without-upload`: files are not uploaded, only hashes are calculated and manifest is created (_optional_, _false_, upload every asset to swarm)

If both `without-scan` and `without-upload` are omitted then `path` is used to associate files, extend the manifest entries, and upload content. 

if `register-names` is set all named nodes.

### Examples
```js
{
   "entries": [
      { 
         "path": "chat",
         "hash": "sdfhsd76ftsd86ft76sdgf78h7tg",
         "status": 200,
         "contentType": "document/pdf"
      },
      ...
   ]
}
```

## Without swarm, the zip fallback

namereg resolution:

`contentOf('eth/wallet') -> 324234kj23h4kj2h3kj423kj4h23`

This name reg has also a `urlOf` where it can find the file (e.g. from a raw pastebin)

It then downloads the file, extracts it and resolves all relative/absolute paths, based on the manifest it finds in it.

For the developer, the upload mechanism in mix will be the same, as he chooses a folder and can provide a `serverconfig.json` (or manfiest)

The only difference is the lookup and where it gets the files from.

```
swarm -> content hashes
before swarm -> zip file content
```

And both are resolved through the same manifest scheme

## Server config examples:

URL: bzz://dsf32f3cdsfsd/somefolder/other
Same as: eth://myname.reggae/somefolder/other

We should also map folder with and without "/" so that the path lookup for path: "/something/myfolder" is the same as "/something/myfolder/"

```js
{
  previous: 'jgjgj67576576576567ytjy',
  first: 'ds564rh5656hhfghfg',
  entries:[{
    // Custom error page
    path: '/i18n/',
    file: '/errorpages/404.html',
    // parses "file" when processing the folder and add: hash: '7685trgdrreewr34f34', contentType: 'text/html'
    status: 404

  },{
    // custom fallback file for this folder: "/images/sdffsdfds/"
    path: '/images/sdffsdfds/',
    file: '/index.html',
    // parses "file" when processing the folder and add: hash: '345678678678678678tryrty', contentType: 'text/html'

  },{
    // custom fallback file with custom header.
    path: '/',
    file: '/index.html',
    // parses "file" when processing the folder and add: hash: '434534534f34k234234hrkj34hkjrh34', contentType: 'text/html'
    status: 500

  },{
    // redirect (changing url after?)
    path: '/somefolder/',
    redirect: 'http://google.com'

  },{
    // linking?
    path: '/somefolder/other/',
    link: 'bzz://43greg45gerg5t45gerge/chat/' // hash to another manifest

  },{
    // downloading a file by pointing to a folder
    path: '/somefolder/other/',
    file: '/mybook.pdf',
    // parses "file" when processing the folder and add: hash: '645325ytrhfgdge4tgre43f34', BUT no contentType, as its already present
    contentType: 'application/octet-stream' // trigger a download in the browser for this link)

  },{
    // downloading
    path: '/test.html',
    file: '/test.html',
    // parses "file" when processing the folder and add: hash: '645325ytrhfgdge4tgre43f34', BUT no contentType, as its already present
    contentType: 'application/octet-stream' // trigger a download in the browser for this link)

  // automatic generated files
  },{
    path: '/i18n/app.en.json',
    hash: '456yrtgfds43534t45',
    contentType: 'text/json',
  },{
    path: '/somefolder/other/image.png',
    hash: '434534534f34khrkj34hkjrh34',
    contentType: 'image/png',
  },{
    path: '/somefolder/other/343242.png',
    hash: '434534534f34k234234hrkj34hkjrh34',
    contentType: 'image/png',
  },{
    path: '/somefold/frau.png',
    hash: 'sdfsdfsdfsdfsdfsdfsd',
    contentType: 'image/png',
  }]
}
```