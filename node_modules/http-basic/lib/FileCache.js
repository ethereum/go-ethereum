'use strict';
exports.__esModule = true;
var fs = require("fs");
var path_1 = require("path");
var crypto_1 = require("crypto");
function jsonParse(data, cb) {
    var result = null;
    try {
        result = JSON.parse(data);
    }
    catch (ex) {
        return cb(ex);
    }
    cb(null, result);
}
function getCacheKey(url) {
    var hash = crypto_1.createHash('sha512');
    hash.update(url);
    return hash.digest('hex');
}
var FileCache = /** @class */ (function () {
    function FileCache(location) {
        this._location = location;
    }
    FileCache.prototype.getResponse = function (url, callback) {
        var key = path_1.resolve(this._location, getCacheKey(url));
        fs.readFile(key + '.json', 'utf8', function (err, data) {
            if (err && err.code === 'ENOENT')
                return callback(null, null);
            else if (err)
                return callback(err, null);
            jsonParse(data, function (err, response) {
                if (err) {
                    return callback(err, null);
                }
                var body = fs.createReadStream(key + '.body');
                response.body = body;
                callback(null, response);
            });
        });
    };
    FileCache.prototype.setResponse = function (url, response) {
        var key = path_1.resolve(this._location, getCacheKey(url));
        var errored = false;
        fs.mkdir(this._location, function (err) {
            if (err && err.code !== 'EEXIST') {
                console.warn('Error creating cache: ' + err.message);
                return;
            }
            response.body.pipe(fs.createWriteStream(key + '.body')).on('error', function (err) {
                errored = true;
                console.warn('Error writing to cache: ' + err.message);
            }).on('close', function () {
                if (!errored) {
                    fs.writeFile(key + '.json', JSON.stringify({
                        statusCode: response.statusCode,
                        headers: response.headers,
                        requestHeaders: response.requestHeaders,
                        requestTimestamp: response.requestTimestamp
                    }, null, '  '), function (err) {
                        if (err) {
                            console.warn('Error writing to cache: ' + err.message);
                        }
                    });
                }
            });
        });
    };
    FileCache.prototype.updateResponseHeaders = function (url, response) {
        var key = path_1.resolve(this._location, getCacheKey(url));
        fs.readFile(key + '.json', 'utf8', function (err, data) {
            if (err) {
                console.warn('Error writing to cache: ' + err.message);
                return;
            }
            var parsed = null;
            try {
                parsed = JSON.parse(data);
            }
            catch (ex) {
                console.warn('Error writing to cache: ' + ex.message);
                return;
            }
            fs.writeFile(key + '.json', JSON.stringify({
                statusCode: parsed.statusCode,
                headers: response.headers,
                requestHeaders: parsed.requestHeaders,
                requestTimestamp: response.requestTimestamp
            }, null, '  '), function (err) {
                if (err) {
                    console.warn('Error writing to cache: ' + err.message);
                }
            });
        });
    };
    FileCache.prototype.invalidateResponse = function (url, callback) {
        var key = path_1.resolve(this._location, getCacheKey(url));
        fs.unlink(key + '.json', function (err) {
            if (err && err.code === 'ENOENT')
                return callback(null);
            else
                callback(err || null);
        });
    };
    return FileCache;
}());
exports["default"] = FileCache;
