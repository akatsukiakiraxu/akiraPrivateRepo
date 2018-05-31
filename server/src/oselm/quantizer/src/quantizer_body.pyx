# distutils: sources = src/quantizer.c
cimport numpy as np

cdef extern from "quantizer.h":
    void quantize(int x_min, int x_max, int x_count, float y_min, float y_max, int y_count, float* input, float* output)

def quantize_body(int x_min, int x_max, int x_count, float y_min, float y_max, int y_count, np.ndarray[float, ndim=1] input, np.ndarray[float, ndim=1] output):
    quantize(x_min, x_max, x_count, y_min, y_max, y_count, &input[0], &output[0])

