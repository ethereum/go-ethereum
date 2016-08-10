(function(window, angular, $) {
    'use strict';
    angular.module('FileManagerApp', ['pascalprecht.translate', 'ngFileUpload']);

    /**
     * jQuery inits
     */
    $(window.document).on('shown.bs.modal', '.modal', function() {
        window.setTimeout(function() {
            $('[autofocus]', this).focus();
        }.bind(this), 100);
    });

    $(window.document).on('click', function() {
        $('#context-menu').hide();
    });

    $(window.document).on('contextmenu', '.main-navigation .table-files tr.item-list:has("td"), .item-list', function(e) {
        var menu = $('#context-menu');

        if (e.pageX >= window.innerWidth - menu.width()) {
            e.pageX -= menu.width();
        }
        if (e.pageY >= window.innerHeight - menu.height()) {
            e.pageY -= menu.height();
        }

        menu.hide().css({
            left: e.pageX,
            top: e.pageY
        }).show();
        e.preventDefault();
    });

    if (! Array.prototype.find) {
        Array.prototype.find = function(predicate) {
            if (this == null) {
                throw new TypeError('Array.prototype.find called on null or undefined');
            }
            if (typeof predicate !== 'function') {
                throw new TypeError('predicate must be a function');
            }
            var list = Object(this);
            var length = list.length >>> 0;
            var thisArg = arguments[1];
            var value;

            for (var i = 0; i < length; i++) {
                value = list[i];
                if (predicate.call(thisArg, value, i, list)) {
                    return value;
                }
            }
            return undefined;
        };
    }
 
})(window, angular, jQuery);
