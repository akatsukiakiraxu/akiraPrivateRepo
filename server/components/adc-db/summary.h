#ifndef SUMMARY_H_INC_
#define SUMMARY_H_INC_

#include <stdint.h>
#include <stdio.h>

#include "adc-db.h"
#include "sensor.h"

void write_data_summary(FILE* fp, int r,
	uint16_t *read_buf[],
	int16_t *fft_buf[],
	const uint16_t *expected,
	const uint16_t *fft_expected);

void init_summarizer(void);
void dispose_summarizer(void);
void write_trigger_summary(const struct db_metadata *metadata);

#endif /* SUMMARY_H_INC_ */
