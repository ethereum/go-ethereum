(function (angular, $) {
    'use strict';
    angular.module('FileManagerApp').service('apiHandler', ['$http', '$q', '$window', '$translate', 'Upload',
        function ($http, $q, $window, $translate, Upload) {

            $http.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest';

            var ApiHandler = function () {
                this.inprocess = false;
                this.asyncSuccess = false;
                this.error = '';
                this.rootJson = '';
                this.mainScope = {};

                this.fullManifest = {};
                this.manifestQueue = [];
                this.swarmHash = '';
            };

            ApiHandler.prototype.buildSwarmTree = function (paths) {
                var result = {};

                function fixPath(path) {
                    if (path != '/' && path.slice(0, 1) != '/') {
                        path = '/' + path.substring(0, path.length);
                    }

                    if (path != '/' && path.slice(-1) == '/') {
                        path = path.substring(0, path.length - 1);
                    }

                    return path;
                }

                function initPath(path) {
                    result[path] = {
                        items: [],
                        paths: []
                    };
                }

                var keyPaths = Object.keys(paths);
                keyPaths.forEach(function (path) {
                    var currentPath = [];
                    var parts = path.split('/');
                    var myPath = currentPath.join('');
                    if (!myPath) {
                        myPath = '/';
                    }

                    if (!result[myPath]) {
                        initPath(myPath);
                    }

                    var fileName = parts[parts.length - 1];
                    if (fileName.indexOf('.') !== -1) {
                        delete parts[parts.length - 1];
                        var fileRoot = parts.join('/');
                        if (!fileRoot) {
                            fileRoot = '/';
                        }

                        fileRoot = fixPath(fileRoot);
                        if (!result[fileRoot]) {
                            initPath(fileRoot);
                        }

                        result[fileRoot].items.push({
                            "time": "07:09",
                            "day": "7",
                            "month": "Jun",
                            "size": "4096",
                            "group": "860",
                            "user": "igor.shadurin@gmail.com",
                            "number": "6",
                            "rights": "drwxr-xr-x",
                            "type": "file",
                            "realName": paths[path],
                            "name": fileName,
                            "date": "2016-06-07 09:21:40"
                        });

                    }

                    var folderPath = [];
                    parts.forEach(function (part) {
                        var myFolderPath = folderPath.join('/');
                        if (!myFolderPath) {
                            myFolderPath = '/';
                        }

                        myFolderPath = fixPath(myFolderPath);

                        if (!result[myFolderPath]) {
                            initPath(myFolderPath);
                        }

                        // there are paths
                        if (!result[myFolderPath].paths[part]) {
                            result[myFolderPath].paths[part] = 1;
                            result[myFolderPath].items.push({
                                "time": "07:09",
                                "day": "7",
                                "month": "Jun",
                                "size": "4096",
                                "group": "860",
                                "user": "igor.shadurin@gmail.com",
                                "number": "6",
                                "rights": "drwxr-xr-x",
                                "type": "dir",
                                "realName": part,
                                "name": part,
                                "date": "2016-06-07 09:21:40"
                            });
                        }

                        currentPath.push(part);
                        folderPath.push(part);
                    });
                });

                return result;
            };

            ApiHandler.prototype.fixUrl = function (url) {
                function getCookie(name) {
                    var matches = document.cookie.match(new RegExp(
                        "(?:^|; )" + name.replace(/([\.$?*|{}\(\)\[\]\\\/\+^])/g, '\\$1') + "=([^;]*)"
                    ));
                    return matches ? decodeURIComponent(matches[1]) : undefined;
                }

                // if is_debug enabled you can use swarm via php proxy
                if (getCookie('is_debug') == 1) {
                    url = url.replace('/bzz:', '/proxy/bzz.php').replace('/bzzr:', '/proxy/bzzr.php');
                }

                console.log(url);

                return url;
            };

            ApiHandler.prototype.deferredHandler = function (data, deferred, defaultMsg) {
                if (!data || typeof data !== 'object') {
                    this.error = 'Bridge response error, please check the docs';
                }
                if (data.result && data.result.error) {
                    this.error = data.result.error;
                }
                if (!this.error && data.error) {
                    this.error = data.error.message;
                }
                if (!this.error && defaultMsg) {
                    this.error = defaultMsg;
                }
                if (this.error) {
                    return deferred.reject(data);
                }
                return deferred.resolve(data);
            };

            ApiHandler.prototype.list = function (apiUrl, path, customDeferredHandler) {
                var self = this;
                var dfHandler = customDeferredHandler || self.deferredHandler;
                var deferred = $q.defer();

                console.log("path " + path);

                self.inprocess = true;
                self.error = '';
                var data = {};
                data.result = self.mainScope.swarmTree[path].items;
                dfHandler(data, deferred);
                self.inprocess = false;
                return deferred.promise;
            };

            ApiHandler.prototype.copy = function (apiUrl, items, path) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'copy',
                    items: items,
                    newPath: path
                };

                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_copying'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.move = function (apiUrl, items, path) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'move',
                    items: items,
                    newPath: path
                };
                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_moving'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.remove = function (apiUrl, items) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'remove',
                    items: items
                };

                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_deleting'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.upload = function (apiUrl, destination, files) {
                var self = this;
                var deferred = $q.defer();
                self.inprocess = true;
                self.progress = 0;
                self.error = '';

                var data = {
                    destination: destination
                };

                for (var i = 0; i < files.length; i++) {
                    data['file-' + i] = files[i];
                }

                if (files && files.length) {
                    Upload.upload({
                        url: apiUrl,
                        data: data
                    }).then(function (data) {
                        self.deferredHandler(data, deferred);
                    }, function (data) {
                        self.deferredHandler(data, deferred, 'Unknown error uploading files');
                    }, function (evt) {
                        self.progress = Math.min(100, parseInt(100.0 * evt.loaded / evt.total)) - 1;
                    })['finally'](function () {
                        self.inprocess = false;
                        self.progress = 0;
                    });
                }

                return deferred.promise;
            };

            ApiHandler.prototype.getContent = function (apiUrl, itemPath) {
                var self = this;
                var deferred = $q.defer();
                self.inprocess = true;
                self.error = '';

                var url = this.fixUrl('/bzz:/' + this.swarmHash + itemPath);

                $http.get(url, {}).success(function (data) {
                    self.deferredHandler({"result": data}, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_getting_content'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.edit = function (apiUrl, itemPath, content) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'edit',
                    item: itemPath,
                    content: content
                };

                self.inprocess = true;
                self.error = '';

                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_modifying'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.rename = function (apiUrl, itemPath, newPath) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'rename',
                    item: itemPath,
                    newItemPath: newPath
                };
                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_renaming'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.getUrl = function (apiUrl, path) {
                return "files" + path;
            };

            ApiHandler.prototype.download = function (apiUrl, itemPath, toFilename, downloadByAjax, forceNewWindow, swarmTree) {
                var url = this.fixUrl('/bzz:/' + this.swarmHash + itemPath);

                return !!$window.open(url, '_blank', '');
            };

            ApiHandler.prototype.downloadMultiple = function (apiUrl, items, toFilename, downloadByAjax, forceNewWindow) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'downloadMultiple',
                    items: items,
                    toFilename: toFilename
                };
                var url = [apiUrl, $.param(data)].join('?');

                if (!downloadByAjax || forceNewWindow || !$window.saveAs) {
                    !$window.saveAs && $window.console.error('Your browser dont support ajax download, downloading by default');
                    return !!$window.open(url, '_blank', '');
                }

                self.inprocess = true;
                $http.get(apiUrl).success(function (data) {
                    var bin = new $window.Blob([data]);
                    deferred.resolve(data);
                    $window.saveAs(bin, toFilename);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_downloading'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.compress = function (apiUrl, items, compressedFilename, path) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'compress',
                    items: items,
                    destination: path,
                    compressedFilename: compressedFilename
                };

                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_compressing'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.extract = function (apiUrl, item, folderName, path) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'extract',
                    item: item,
                    destination: path,
                    folderName: folderName
                };

                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_extracting'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.changePermissions = function (apiUrl, items, permsOctal, permsCode, recursive) {
                var self = this;
                var deferred = $q.defer();
                var data = {
                    action: 'changePermissions',
                    items: items,
                    perms: permsOctal,
                    permsCode: permsCode,
                    recursive: !!recursive
                };

                self.inprocess = true;
                self.error = '';
                $http.post(apiUrl, data).success(function (data) {
                    self.deferredHandler(data, deferred);
                }).error(function (data) {
                    self.deferredHandler(data, deferred, $translate.instant('error_changing_perms'));
                })['finally'](function () {
                    self.inprocess = false;
                });
                return deferred.promise;
            };

            ApiHandler.prototype.readFile = function (file, onComplete) {
                var reader = new FileReader();
                reader.onload = function (evt) {
                    if (onComplete) {
                        onComplete(evt.target.result)
                    }
                };
                reader.readAsArrayBuffer(file);
            };

            ApiHandler.prototype.createFolder = function (apiUrl, path) {
                var self = this;
                var deferred = $q.defer();
                console.log(path);
                // /swarm:/f44327c9a9b5b3723083bf601a4c4607490541c94b3fe84ee0cb19b65f418628/6666fsghsfg
                path = '/swarm:/' + this.swarmHash + path;
                var exploded = path.split('/');
                exploded.shift();
                if (exploded.length != 3) {
                    console.log('Oh, length must be 3');
                    return;
                }

                if (exploded[0] == 'swarm:' && exploded[1].length == 64 && exploded[2].length > 0) {
                    var hash = exploded[1];
                    var newFolderName = exploded[2];
                } else {
                    console.log('Oh, so bad params');
                    return;
                }

                // todo get file for root folder
                // put it to current hash
                var putUrlIndex = '/bzz:/' + hash + '/' + newFolderName + '/';
                putUrlIndex = this.fixUrl(putUrlIndex);

                self.inprocess = true;
                self.error = '';

                // todo check this variants:
                //http://stackoverflow.com/questions/25152700/angularjs-put-binary-data-from-arraybuffer-to-the-server
                //https://uncorkedstudios.com/blog/multipartformdata-file-upload-with-angularjs

                // todo angular file uploading:
                // https://github.com/danialfarid/ng-file-upload#usage
                //http://jsfiddle.net/danialfarid/maqbzv15/1118/
                if ($('#newFolderFile').prop('files').length) {
                    var isUploadFileToRoot = $('#uploadFileToRoot').is(':checked');
                    var file = $('#newFolderFile').prop('files')[0];
                    var fileName = file.name;
                    $http.put(putUrlIndex, file).success(function (data) {
                        console.log("putUrlIndex answer is " + data);
                        // todo reload files list
                        window.location = "/#/" + data;
                        if (isUploadFileToRoot) {
                            var putUrlFile = '/bzz:/' + data + '/' + newFolderName + '/' + fileName;
                            putUrlFile = self.fixUrl(putUrlFile);
                            $http.put(putUrlFile, file).success(function (data) {
                                console.log("putUrlFile answer is " + data);
                                window.location = "/#/" + data;
                                self.deferredHandler(data, deferred);
                            }).error(function (data) {
                                self.deferredHandler(data, deferred, $translate.instant('error_creating_folder'));
                            })['finally'](function () {
                                self.inprocess = false;
                            });
                        } else {
                            self.deferredHandler(data, deferred);
                        }
                    }).error(function (data) {
                        self.deferredHandler(data, deferred, $translate.instant('error_creating_folder'));
                    })['finally'](function () {
                        self.inprocess = false;
                    });
                } else {
                    alert('Select file and try again');
                }

                return deferred.promise;
            };

            ApiHandler.prototype.downloadFullManifest = function (hash, key, onFinally) {
                var self = this;
                self.inprocess = true;
                self.manifestQueue.push("lol");

                if (!key) {
                    key = '';
                }

                var url = this.fixUrl('/bzzr:/') + hash;
                $http.get(url, {}).success(function (data) {
                    $.each(data.entries, function (k, v) {
                        var path = v.path;
                        if (!path) {
                            // handle root folder hash here
                        }

                        var keyPath = key + path;
                        self.fullManifest[keyPath] = v.hash;
                        if (v.contentType == "application/bzz-manifest+json") {
                            delete self.fullManifest[keyPath];
                            self.downloadFullManifest(v.hash, keyPath, onFinally);
                        }
                    });
                }).error(function (data) {
                    //dfHandler(data, deferred, 'Unknown error listing, check the response');
                })['finally'](function () {
                    self.manifestQueue.pop();
                    if (self.manifestQueue.length == 0) {
                        self.inprocess = false;
                        if (onFinally) {
                            onFinally();
                        }
                    }
                });
            };

            return ApiHandler;

        }]);
})(angular, jQuery);