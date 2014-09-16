// this function is included locally, but you can also include separately via a header definition
function request(url, callback) {
	var xhr = new XMLHttpRequest();
	xhr.onreadystatechange = (function(req) {
		return function() {
			if(req.readyState === 4) {
				callback(req);
			}
		}
	})(xhr);
	xhr.open('GET', url, true);
	xhr.send('');
}
