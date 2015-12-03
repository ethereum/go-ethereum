/*
---
description: Swipe events for touch devices.

license: MIT-style.

authors:
- Caleb Troughton

requires:
	core/1.2.4:
	- Element.Event
	- Class
	- Class.Extras

provides:
	MooSwipe
*/
var MooSwipe = MooSwipe || new Class({
	Implements: [Options, Events],

	options: {
		//onSwipeleft: $empty,
		//onSwiperight: $empty,
		//onSwipeup: $empty,
		//onSwipedown: $empty,
		tolerance: 50,
		preventDefaults: true
	},

	element: null,
	startX: null,
	startY: null,
	isMoving: false,

	initialize: function(el, options) {
		this.setOptions(options);
		this.element = $(el);
		this.element.addListener('touchstart', this.onTouchStart.bind(this));
	},

	cancelTouch: function() {
		this.element.removeListener('touchmove', this.onTouchMove);
		this.startX = null;
		this.startY = null;
		this.isMoving = false;
	},

	onTouchMove: function(e) {
		if (e.touches.length == 1) {
		    this.options.preventDefaults && e.preventDefault();
		    if (this.isMoving) {
			var dx = this.startX - e.touches[0].pageX;
			var dy = this.startY - e.touches[0].pageY;
			if (Math.abs(dx) >= this.options.tolerance) {
				this.cancelTouch();
				this.fireEvent(dx > 0 ? 'swipeleft' : 'swiperight');
			} else if (Math.abs(dy) >= this.options.tolerance) {
				this.cancelTouch();
				this.fireEvent(dy > 0 ? 'swipedown' : 'swipeup');
			}
		    }
		}
		else if (this.isMoving) {
		  this.cancelTouch();
		}
	},

	onTouchStart: function(e) {
		if (e.touches.length == 1) {
			this.startX = e.touches[0].pageX;
			this.startY = e.touches[0].pageY;
			this.isMoving = true;
			this.element.addListener('touchmove', this.onTouchMove.bind(this));
		}
	}
});
