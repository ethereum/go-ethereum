/*
---
description: Determines when the user is idle (not interacting with the page) so that you can respond appropriately.

license:
- MIT-style license

authors:
- Espen 'Rexxars' Hovlandsdal (http://rexxars.com/)

requires:
core/1.2.4:
- Class.Extras
- Element.Event

provides:
- IdleTimer

inspiration:
- Inspired by Nicholas C. Zakas' Idle Timer (http://yuilibrary.com/gallery/show/idletimer) Copyright (c) 2009 Nicholas C. Zakas, [YUI BSD](http://developer.yahoo.com/yui/license.html)
- Also inspired by Paul Irish's jQuery idleTimer (http://paulirish.com/2009/jquery-idletimer-plugin/) Copyright (c) 2009 Paul Irish, [MIT License](http://opensource.org/licenses/mit-license.php)
...
*/

IdleTimer = new Class({

	Implements: [Events, Options],
	
	options: {
		/*
		onStart: function(){},
		onStop: function(){},
		onIdle: function(){},
		onActive: function(){},
		onTimeoutChanged: function(){},
		*/
		timeout: 60000,
		events: ['mousemove', 'keydown', 'mousewheel', 'mousedown', 'touchstart', 'touchmove']
	},
	
	initialize: function(element, options) {
		this.setOptions(options);
		this.element = document.id(element);
		this.activeBound = this.active.bind(this);
		this.isIdle = false;
		this.started = false;
		this.lastPos = false;
	},
	
	/**
	 * Stops any existing timeouts and removes the bound events
	 */
	stop: function() {
		clearTimeout(this.timer);
		
		// Remove bound events
		for(var i = 0; i < this.options.events.length; i++) {
			this.element.removeEvent(this.options.events[i], this.activeBound);
		}
		this.bound = false;
		this.started = false;
		this.lastPos = false;
		this.fireEvent('stop');
		return this;
	},
	
	/**
	 * Triggered when the user becomes active. May also be launched manually by scripts
	 * if implementing some sort of custom events etc. An example would be flash files
	 * which does not trigger the documents onmousemove event, you could have the flash
	 * call this method to prevent the idle event from being triggered.
	 */
	active: function(e) {
		if(e.event.type == 'mousemove')
		{
		  // Fix https://code.google.com/p/chromium/issues/detail?id=103041
		  var pos = [e.event.clientX, e.event.clientY];
		  if(this.lastPos === false ||
		      (this.lastPos[0] != pos[0] && this.lastPos[1] != pos[1]))
		    this.lastPos = pos;
		  else
		    return;
		}
		clearTimeout(this.timer);
		if(this.isIdle) this.fireEvent('active');
		this.isIdle = false;
		this.start();
	},
	
	/**
	 * Fired when the user becomes idle
	 */
	idle: function() {
		if(this.timer) clearTimeout(this.timer); // If called manually, timer will have to be removed
		this.isIdle = true;
		this.fireEvent('idle');
	},
	
	/**
	 * Starts the timer which eventually will reach idle() if the user is inactive
	 */
	start: function() {
		if(this.timer) clearTimeout(this.timer); // If called twice, timer will have to be removed
		this.timer = this.idle.delay(this.options.timeout, this);
		this.lastActive = Date.now();
		if(!this.bound) this.bind();
		this.started = true;
		return this;
	},
	
	/**
	 * Bind events to the element
	 */
	bind: function() {
		for(var i = 0; i < this.options.events.length; i++) {
			this.element.addEvent(this.options.events[i], this.activeBound);
		}
		this.bound = true;
		this.fireEvent('start');
	},
	
	/**
	 * Returns how many seconds/milliseconds have passed since the user was last idle
	 */
	getIdleTime: function(returnSeconds) {
		return returnSeconds ? Math.round((Date.now() - this.lastActive) / 1000) : Date.now() - this.lastActive;
	},
	
	/**
	 * Sets the number of milliseconds is concidered "idle".
	 * Will also attempt to fix any difference in the old and new timeout values,
	 * unless you pass true as whenActive - in this case the new timeout will be
	 * in play the next time the user is active again.
	 */
	setTimeout: function(newTime, whenActive) {
		var old = this.options.timeout;
		this.options.timeout = newTime;
		
		if(whenActive) return this; // The developer wants to wait until the next time the user is active before setting the new timeout
		
		// In all cases, we need a new timer
		clearTimeout(this.timer);
		
		// Fire a new timeout event
		this.fireEvent('timeoutChanged', newTime);
		
		// How much time has ellapsed since we were last active?
		var elapsed = this.getIdleTime();
		
		if(elapsed < newTime && this.isIdle) {
			// Set as active, cause the new "idle" time has not been reached
			this.fireEvent('active');
			this.isIdle = false;
		} else if(elapsed >= newTime) {
			// We've not reached the limit before, but with the new timeout, we now have.
			this.idle();
			return this;
		}
		
		// Set new timer
		this.timer = this.idle.delay(newTime - elapsed, this);
		return this;
	}

});

Element.Properties.idle = {

	set: function(options) {
		var idle = this.retrieve('idle');
		if (idle) idle.stop();
		return this.eliminate('idle').store('idle:options', options);
	},

	get: function(options) {
		if (options || !this.retrieve('idle')) {
			if (options || !this.retrieve('idle:options')) this.set('idle', options);
			this.store('idle', new IdleTimer(this, this.retrieve('idle:options')));
		}
		return this.retrieve('idle');
	}

};

Element.Events.idle = {
	onAdd: function(fn) {
		var global = this.get ? false : true;
		var idler = global ? window.idleTimer : this.get('idle');
		if(global && !idler) { idler = window.idleTimer = new IdleTimer(Browser.ie ? document : this); }
		if(!idler.started) idler.start();
		idler.addEvent('idle', fn);
	}
};
Element.Events.active = {
	onAdd: function(fn) {
		(this.get ? this.get('idle') : window.idleTimer).addEvent('active', fn);
	}
};
