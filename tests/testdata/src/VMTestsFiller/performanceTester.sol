contract PerformanceTester {	
	function ackermann(uint m, uint n) returns (uint) {
		if (m == 0)
			return n + 1;
		
		if (n == 0)
			return ackermann(m - 1, 1);
		
		return ackermann(m - 1, ackermann(m, n - 1));
	}
	
	function fibonacci(uint n) returns (uint) {
	    if (n == 0 || n == 1)
	        return n;
	    return fibonacci(n - 1) + fibonacci(n - 2);
	}
}