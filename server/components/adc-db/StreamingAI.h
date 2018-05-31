#ifndef _STREAMINGAI_H_
#define _STREAMINGAI_H_

#include <stddef.h>
#include <stdlib.h>
#include <stdio.h>

//-----------------------------------------------------------------------------------
// define the type of callback function
//-----------------------------------------------------------------------------------]
typedef void (* trigger_handler_t)(int , int , int , int);
typedef void (* data_handler_t)(const void *data, size_t length, const void *fft_data, size_t fft_length);
#define CHK_RESULT(ret) {if(BioFailed(ret))break;}

#endif /* _STREAMINGAI_H_ */
