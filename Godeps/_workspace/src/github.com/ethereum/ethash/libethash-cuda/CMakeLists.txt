find_package(CUDA)
 
# Pass options to NVCC

 
if (CUDA_FOUND)
set(CUDA_NVCC_FLAGS "   -gencode;arch=compute_30,code=sm_30;
                        -gencode;arch=compute_20,code=sm_20;
                        -gencode;arch=compute_11,code=sm_11;
                        -gencode;arch=compute_12,code=sm_12;
                        -gencode;arch=compute_13,code=sm_13;")
cuda_add_executable(
    ethash-cuda
    libethash.cu)
endif()