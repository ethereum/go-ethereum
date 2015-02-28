#include "cuPrintf.cu"
#include <stdio.h>

__global__ void device_greetings(void)
{
	cuPrintf("Hello, world from the device!\n");
}

int main(void)
{
	// greet from the host
	printf("Hello, world from the host!\n");

	// initialize cuPrintf
	cudaPrintfInit();

	// launch a kernel with a single thread to greet from the device
	device_greetings<<<1,1>>>();

	// display the device's greeting
	cudaPrintfDisplay();

	// clean up after cuPrintf
	cudaPrintfEnd();

	return 0;
}
