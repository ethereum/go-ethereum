(function(angular) {
    'use strict';
    var app = angular.module('FileManagerApp');

    app.filter('strLimit', ['$filter', function($filter) {
        return function(input, limit) {
            if (input.length <= limit) {
                return input;
            }
            return $filter('limitTo')(input, limit) + '...';
        };
    }]);

    app.filter('formatDate', ['$filter', function() {
        return function(input) {
            return input instanceof Date ?
                input.toISOString().substring(0, 19).replace('T', ' ') :
                (input.toLocaleString || input.toString).apply(input);
        };
    }]);

    app.filter('humanReadableFileSize', ['$filter', 'fileManagerConfig', function($filter, fileManagerConfig) {
      // See https://en.wikipedia.org/wiki/Binary_prefix
      var decimalByteUnits = [' kB', ' MB', ' GB', ' TB', 'PB', 'EB', 'ZB', 'YB'];
      var binaryByteUnits = ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];

      return function(input) {
        var i = -1;
        var fileSizeInBytes = input;

        do {
          fileSizeInBytes = fileSizeInBytes / 1024;
          i++;
        } while (fileSizeInBytes > 1024);

        var result = fileManagerConfig.useBinarySizePrefixes ? binaryByteUnits[i] : decimalByteUnits[i];
        return Math.max(fileSizeInBytes, 0.1).toFixed(1) + ' ' + result;
      };
    }]);
})(angular);
