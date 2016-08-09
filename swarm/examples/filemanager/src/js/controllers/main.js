(function (angular, $) {
    'use strict';
    angular.module('FileManagerApp').controller('FileManagerCtrl', [
        '$scope', '$rootScope', '$window', '$translate', 'fileManagerConfig', 'item', 'fileNavigator', 'apiMiddleware',
        function ($scope, $rootScope, $window, $translate, fileManagerConfig, Item, FileNavigator, ApiMiddleware) {

            var $storage = $window.localStorage;
            $scope.config = fileManagerConfig;
            $scope.reverse = false;
            $scope.predicate = ['model.type', 'model.name'];
            $scope.order = function (predicate) {
                $scope.reverse = ($scope.predicate[1] === predicate) ? !$scope.reverse : false;
                $scope.predicate[1] = predicate;
            };
            $scope.query = '';
            $scope.fileNavigator = new FileNavigator();
            $scope.apiMiddleware = new ApiMiddleware();
            $scope.uploadFileList = [];
            $scope.viewTemplate = $storage.getItem('viewTemplate') || 'main-table.html';
            $scope.fileList = [];
            $scope.temps = [];
            $scope.swarmHash = '';
            $scope.swarmTree = {
                "/": {
                    items: [],
                    paths: []
                }
            };

            $scope.$watch('temps', function () {
                if ($scope.singleSelection()) {
                    $scope.temp = $scope.singleSelection();
                } else {
                    $scope.temp = new Item({rights: 644});
                    $scope.temp.multiple = true;
                }
                $scope.temp.revert();
            });

            $scope.fileNavigator.onRefresh = function () {
                $scope.temps = [];
                $rootScope.selectedModalPath = $scope.fileNavigator.currentPath;
            };

            $scope.setTemplate = function (name) {
                $storage.setItem('viewTemplate', name);
                $scope.viewTemplate = name;
            };

            $scope.changeLanguage = function (locale) {
                if (locale) {
                    $storage.setItem('language', locale);
                    return $translate.use(locale);
                }
                $translate.use($storage.getItem('language') || fileManagerConfig.defaultLang);
            };

            $scope.isSelected = function (item) {
                return $scope.temps.indexOf(item) !== -1;
            };

            $scope.selectOrUnselect = function (item, $event) {
                var indexInTemp = $scope.temps.indexOf(item);
                var isRightClick = $event && $event.which == 3;

                if ($event && $event.target.hasAttribute('prevent')) {
                    $scope.temps = [];
                    return;
                }
                if (!item || (isRightClick && $scope.isSelected(item))) {
                    return;
                }
                if ($event && $event.shiftKey && !isRightClick) {
                    var list = $scope.fileList;
                    var indexInList = list.indexOf(item);
                    var lastSelected = $scope.temps[0];
                    var i = list.indexOf(lastSelected);
                    var current = undefined;
                    if (lastSelected && list.indexOf(lastSelected) < indexInList) {
                        $scope.temps = [];
                        while (i <= indexInList) {
                            current = list[i];
                            !$scope.isSelected(current) && $scope.temps.push(current);
                            i++;
                        }
                        return;
                    }
                    if (lastSelected && list.indexOf(lastSelected) > indexInList) {
                        $scope.temps = [];
                        while (i >= indexInList) {
                            current = list[i];
                            !$scope.isSelected(current) && $scope.temps.push(current);
                            i--;
                        }
                        return;
                    }
                }
                if ($event && $event.ctrlKey && !isRightClick) {
                    $scope.isSelected(item) ? $scope.temps.splice(indexInTemp, 1) : $scope.temps.push(item);
                    return;
                }
                $scope.temps = [item];
            };

            $scope.singleSelection = function () {
                return $scope.temps.length === 1 && $scope.temps[0];
            };

            $scope.totalSelecteds = function () {
                return {
                    total: $scope.temps.length
                };
            };

            $scope.selectionHas = function (type) {
                return $scope.temps.find(function (item) {
                    return item && item.model.type === type;
                });
            };

            $scope.prepareNewFolder = function () {
                var item = new Item(null, $scope.fileNavigator.currentPath);
                $scope.temps = [item];
                return item;
            };

            $scope.smartClick = function (item) {
                if (item.isFolder()) {
                    return $scope.fileNavigator.folderClick(item);
                }
                if (item.isImage()) {
                    if ($scope.config.previewImagesInModal) {
                        return $scope.openImagePreview(item);
                    }
                    return $scope.apiMiddleware.download(item, true);
                }
                if (item.isEditable()) {
                    return $scope.openEditItem(item);
                }
            };

            $scope.openImagePreview = function () {
                var item = $scope.singleSelection();
                $scope.apiMiddleware.apiHandler.inprocess = true;
                var url = $scope.apiMiddleware.getUrl(item);
                // todo check url for emulator
                url = url.replace('files/', '/bzz:/' + $scope.swarmHash + '/');
                $scope.modal('imagepreview', null, true)
                    .find('#imagepreview-target')
                    .attr('src', url)
                    .unbind('load error')
                    .on('load error', function () {
                        $scope.apiMiddleware.apiHandler.inprocess = false;
                        $scope.$apply();
                    });
            };

            $scope.openEditItem = function () {
                var item = $scope.singleSelection();
                $scope.apiMiddleware.getContent(item).then(function (data) {
                    item.tempModel.content = item.model.content = data.result;
                });
                $scope.modal('edit');
            };

            $scope.modal = function (id, hide, returnElement) {
                var element = $('#' + id);
                element.modal(hide ? 'hide' : 'show');
                $scope.apiMiddleware.apiHandler.error = '';
                $scope.apiMiddleware.apiHandler.asyncSuccess = false;
                return returnElement ? element : true;
            };

            $scope.modalWithPathSelector = function (id) {
                $rootScope.selectedModalPath = $scope.fileNavigator.currentPath;
                return $scope.modal(id);
            };

            $scope.isInThisPath = function (path) {
                var currentPath = $scope.fileNavigator.currentPath.join('/');
                return currentPath.indexOf(path) !== -1;
            };

            $scope.edit = function () {
                $scope.apiMiddleware.edit($scope.singleSelection()).then(function () {
                    $scope.modal('edit', true);
                });
            };

            $scope.changePermissions = function () {
                $scope.apiMiddleware.changePermissions($scope.temps, $scope.temp).then(function () {
                    $scope.modal('changepermissions', true);
                });
            };

            $scope.download = function () {
                var item = $scope.singleSelection();
                if ($scope.selectionHas('dir')) {
                    return;
                }
                if (item) {
                    return $scope.apiMiddleware.download(item, true, $scope.swarmTreeFull);
                }
                return $scope.apiMiddleware.downloadMultiple($scope.temps);
            };

            $scope.copy = function () {
                var item = $scope.singleSelection();
                if (item) {
                    var name = item.tempModel.name.trim();
                    var nameExists = $scope.fileNavigator.fileNameExists(name);
                    if (nameExists && validateSamePath(item)) {
                        $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                        return false;
                    }
                    if (!name) {
                        $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                        return false;
                    }
                }
                $scope.apiMiddleware.copy($scope.temps, $rootScope.selectedModalPath).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.modal('copy', true);
                });
            };

            $scope.compress = function () {
                var name = $scope.temp.tempModel.name.trim();
                var nameExists = $scope.fileNavigator.fileNameExists(name);

                if (nameExists && validateSamePath($scope.temp)) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                    return false;
                }
                if (!name) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                    return false;
                }

                $scope.apiMiddleware.compress($scope.temps, name, $rootScope.selectedModalPath).then(function () {
                    $scope.fileNavigator.refresh();
                    if (!$scope.config.compressAsync) {
                        return $scope.modal('compress', true);
                    }
                    $scope.apiMiddleware.apiHandler.asyncSuccess = true;
                }, function () {
                    $scope.apiMiddleware.apiHandler.asyncSuccess = false;
                });
            };

            $scope.extract = function () {
                var item = $scope.temp;
                var name = $scope.temp.tempModel.name.trim();
                var nameExists = $scope.fileNavigator.fileNameExists(name);

                if (nameExists && validateSamePath($scope.temp)) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                    return false;
                }
                if (!name) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                    return false;
                }

                $scope.apiMiddleware.extract(item, name, $rootScope.selectedModalPath).then(function () {
                    $scope.fileNavigator.refresh();
                    if (!$scope.config.extractAsync) {
                        return $scope.modal('extract', true);
                    }
                    $scope.apiMiddleware.apiHandler.asyncSuccess = true;
                }, function () {
                    $scope.apiMiddleware.apiHandler.asyncSuccess = false;
                });
            };

            $scope.remove = function () {
                $scope.apiMiddleware.remove($scope.temps).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.modal('remove', true);
                });
            };

            $scope.move = function () {
                var anyItem = $scope.singleSelection() || $scope.temps[0];
                if (anyItem && validateSamePath(anyItem)) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_cannot_move_same_path');
                    return false;
                }
                $scope.apiMiddleware.move($scope.temps, $rootScope.selectedModalPath).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.modal('move', true);
                });
            };

            $scope.rename = function () {
                var item = $scope.singleSelection();
                var name = item.tempModel.name;
                var samePath = item.tempModel.path.join('') === item.model.path.join('');
                if (!name || (samePath && $scope.fileNavigator.fileNameExists(name))) {
                    $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                    return false;
                }
                $scope.apiMiddleware.rename(item).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.modal('rename', true);
                });
            };

            $scope.createFolder = function () {
                var item = $scope.singleSelection();
                var name = item.tempModel.name;
                if (!name || $scope.fileNavigator.fileNameExists(name)) {
                    return $scope.apiMiddleware.apiHandler.error = $translate.instant('error_invalid_filename');
                }
                $scope.apiMiddleware.createFolder(item).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.modal('newfolder', true);
                });
            };

            $scope.addForUpload = function ($files) {
                $scope.uploadFileList = $scope.uploadFileList.concat($files);
                $scope.modal('uploadfile');
            };

            $scope.removeFromUpload = function (index) {
                $scope.uploadFileList.splice(index, 1);
            };

            $scope.uploadFiles = function () {
                $scope.apiMiddleware.upload($scope.uploadFileList, $scope.fileNavigator.currentPath).then(function () {
                    $scope.fileNavigator.refresh();
                    $scope.uploadFileList = [];
                    $scope.modal('uploadfile', true);
                }, function (data) {
                    var errorMsg = data.result && data.result.error || $translate.instant('error_uploading_files');
                    $scope.apiMiddleware.apiHandler.error = errorMsg;
                });
            };

            var validateSamePath = function (item) {
                var selectedPath = $rootScope.selectedModalPath.join('');
                var selectedItemsPath = item && item.model.path.join('');
                return selectedItemsPath === selectedPath;
            };

            var getQueryParam = function (param) {
                var found = $window.location.search.substr(1).split('&').filter(function (item) {
                    return param === item.split('=')[0];
                });
                return found[0] && found[0].split('=')[1] || undefined;
            };

            $scope.changeLanguage(getQueryParam('lang'));
            $scope.isWindows = getQueryParam('server') === 'Windows';

            $scope.swarmHash = $scope.apiMiddleware.apiHandler.swarmHash = window.location.hash.substring(2);
            $scope.fileNavigator.apiMiddleware.apiHandler.mainScope = $scope;
            $scope.fileNavigator.apiMiddleware.apiHandler.downloadFullManifest($scope.swarmHash, null, function () {
                var paths = $scope.fileNavigator.apiMiddleware.apiHandler.fullManifest;
                $scope.swarmTreeFull = paths;
                $scope.swarmTree = $scope.fileNavigator.apiMiddleware.apiHandler.buildSwarmTree(paths);
                $scope.fileNavigator.refresh();
            });
        }]);
})(angular, jQuery);
