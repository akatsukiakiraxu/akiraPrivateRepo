#ifndef ADC_DB_H_INC_
#define ADC_DB_H_INC_

#include <stdint.h>
#include <time.h>


struct db_metadata {
    size_t written;
    time_t start_time;
    int overflow;
    uint32_t channel;
    uint32_t code;
    uint32_t palette;
    uint32_t pos;
    uint32_t nfile;
    uint32_t outlier_count;
    uint32_t fft_outlier_count;
    uint32_t status;

};

#endif /* ADC_DB_H_INC_ */
