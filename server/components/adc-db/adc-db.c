#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <time.h>

#include <fcntl.h>
#include <pthread.h>
#include <semaphore.h>
#include <sys/ioctl.h>
#include <sys/stat.h> /* For mkdir */
#include <sys/time.h>
#include <unistd.h>

#include "adc-db.h"
#include "async-io.h"
#ifdef PETALINUX
#include "sensor.h"
#include "summary.h"
#else // x86
#include "StreamingAI.h"
#include "debug.h"
#endif

// #define DEBUG_BIN

#define	DROP_CACHE		/* Enable drop page cache */

/* Uncomment this line if the DB storage is MicroSD. */
/* #define SD_STORAGE */

#define DB_DIR "."

#define	DROP_CACHE_MSG_1	"1\n"	/* Purge the page cache of file data */
#define	DROP_CACHE_MSG_4	"4\n"	/* Disable logging of drop cache */
#define	DROP_CACHE_MSG_LEN	2

#define NUM_FILE_PER_DIR 100000
#define FLUSH_COUNT      100
#define HEADER_SIZE      24
#define MAX_FILE_SIZE    ((size_t)2 * 1024 * 1024 * 1024)

#define CODE_DIR      "code_%02x"
#define PALETTE_DIR   "palette_%02x"
#define POS_DIR       "pos_%02x"
#define DATA_DIR      CODE_DIR "/" PALETTE_DIR "/" POS_DIR
#define DBNUM_FILE    ".dbnum"
#define DB_FILE       "CH%d_%05d.dat"
#define FFT_FILE      "CH%d_%05d-fft.dat"
#define ERROR_FILE    "CH%d_%05d.txt"
#define EXPECTED_FILE "expected.dat"
#define FFT_EXPECTED_FILE "expected-fft.dat"
#define OUTLIER_FILE  "outlier.txt"

#define DEFAULT_CODE    0xff
#define DEFAULT_PALETTE 0xff
#define DEFAULT_POS     0xff

/* Definitions of header field lengths */
#define MAGIC_NUMBER_LENGTH     4
#define HEADER_LENGTH_LENGTH    4
#define DATA_LENGTH_LENGTH      8
#define START_TIME_LENGTH       4
#define FLAGS_LENGTH            4
#define CHANNEL_NUMBER_LENGTH   4
#define CODE_LENGTH             4
#define PALLETE_LENGTH          4
#define POSITION_LENGTH         4
#define RAW_ERROR_COUNT_LENGTH  4
#define FFT_ERROR_COUNT_LENGTH  4

/* Definitions of header field offsets */
#define MAGIC_NUMBER_OFFSET     0
#define HEADER_LENGTH_OFFSET    4
#define DATA_LENGTH_OFFSET      8
#define START_TIME_OFFSET      16
#define FLAGS_OFFSET           20
#define CHANNEL_NUMBER_OFFSET  24
#define CODE_OFFSET            28
#define PALLETE_OFFSET         32
#define POSITION_OFFSET        36
#define RAW_ERROR_COUNT_OFFSET 40
#define FFT_ERROR_COUNT_OFFSET 44

#ifdef PETALINUX
static struct db_metadata db_metadata_list[NUM_CHANNEL];
static struct db_file_data {
	int dbfid;
	int fft_fid;
	FILE *err_fp;
} db_file_data_list[NUM_CHANNEL];
static FILE *outlier_fp = NULL;
static FILE *expfp = NULL;
static FILE *fft_expfp = NULL;

static void init_db(void)
{
	int c;

	mkdir(DB_DIR, S_IRWXU | S_IRWXG | S_IRWXO);
	/* TODO: Implement an error handler */

	outlier_fp = fopen(DB_DIR "/" OUTLIER_FILE, "a");

	for (c = 0; c < NUM_CHANNEL; c++) {
		db_file_data_list[c].dbfid = -1;
		db_file_data_list[c].fft_fid = -1;
	}
}
#else /* x86 */
void start_mic_sensor(char* propertyFileName);
#endif /* #ifdef PETALINUX */

#ifdef PETALINUX
static void create_header(uint8_t *header_buf, const struct db_metadata *metadata)
{
	static const uint8_t magic[4] = { 0xDB, 0xDC, 0x0A, 0x00 };
	static const uint32_t hlen = 40;
	uint64_t dlen = (uint64_t)metadata->written;
	uint32_t t = (uint32_t)metadata->start_time;
	uint32_t flags = metadata->overflow ? 0x00000001 : 0x00000000;

	uint8_t *header_index = header_buf;

	memcpy(header_index, magic, sizeof(magic));
	header_index += sizeof(magic);
	memcpy(header_index, &hlen, sizeof(hlen));
	header_index += sizeof(hlen);
	memcpy(header_index, &dlen, sizeof(dlen));
	header_index += sizeof(dlen);
	memcpy(header_index, &t, sizeof(t));
	header_index += sizeof(t);
	memcpy(header_index, &flags, sizeof(flags));
	header_index += sizeof(flags);
	memcpy(header_index, &metadata->channel, sizeof(metadata->channel));
	header_index += sizeof(metadata->channel);
	memcpy(header_index, &metadata->code, sizeof(metadata->code));
	header_index += sizeof(metadata->code);
	memcpy(header_index, &metadata->palette, sizeof(metadata->palette));
	header_index += sizeof(metadata->palette);
	memcpy(header_index, &metadata->pos, sizeof(metadata->pos));
	header_index += sizeof(metadata->pos);
	memcpy(header_index, &metadata->outlier_count, sizeof(metadata->outlier_count));
	header_index += sizeof(metadata->outlier_count);
	memcpy(header_index, &metadata->fft_outlier_count, sizeof(metadata->fft_outlier_count));
	header_index += sizeof(metadata->fft_outlier_count);
}

static void write_header(int fid, const struct db_metadata *metadata)
{
	uint8_t *header_buf = async_file_malloc(48);
	create_header(header_buf, metadata);
	async_file_rewind_and_write(fid, header_buf, 48);
}

static void load_nfile(void)
{
	char name[256];
	FILE *fp;

	int code = db_metadata_list[0].code;
	int palette = db_metadata_list[0].palette;
	int pos = db_metadata_list[0].pos;
        int ret = 0;

	sprintf(name, DB_DIR "/" CODE_DIR, code);
	mkdir(name, S_IRWXU | S_IRWXG | S_IRWXO);
	sprintf(name, DB_DIR "/" CODE_DIR "/" PALETTE_DIR, code, palette);
	mkdir(name, S_IRWXU | S_IRWXG | S_IRWXO);
	sprintf(name, DB_DIR "/" CODE_DIR "/" PALETTE_DIR "/" POS_DIR, code, palette, pos);
	mkdir(name, S_IRWXU | S_IRWXG | S_IRWXO);

	sprintf(name, DB_DIR "/" DATA_DIR "/" DBNUM_FILE, code, palette, pos);

	fp = fopen(name, "r");
	if (fp == NULL) {
		db_metadata_list[0].nfile = 0;
		db_metadata_list[1].nfile = 0;
		db_metadata_list[2].nfile = 0;
		db_metadata_list[3].nfile = 0;
	} else {
		ret = fscanf(fp, "%d %d %d %d",
			&db_metadata_list[0].nfile,
			&db_metadata_list[1].nfile,
			&db_metadata_list[2].nfile,
			&db_metadata_list[3].nfile);
		fclose(fp);
	}
	fp = fopen(name, "w");
	if (fp == NULL) {
		perror("Failed to open .dbnum file for write.");
	}
	fprintf(fp, "%d %d %d %d",
		db_metadata_list[0].nfile + 1,
		db_metadata_list[1].nfile + 1,
		db_metadata_list[2].nfile + 1,
		db_metadata_list[3].nfile + 1);
	fclose(fp);
}

static void open_datafile(struct db_metadata *metadata, struct db_file_data *fdata)
{
	char name[256];

	int code = metadata->code;
	int palette = metadata->palette;
	int pos = metadata->pos;
	uint32_t ch = metadata->channel;

	if (metadata->nfile >= NUM_FILE_PER_DIR) {
		fdata->dbfid = -1;
		fdata->fft_fid = -1;
		return;
	}

	sprintf(name, DB_DIR "/" DATA_DIR "/" DB_FILE, code, palette, pos, (int)ch + 1, metadata->nfile);
	fdata->dbfid = async_file_open(name);
	if (fdata->dbfid < 0) {
		perror("Failed to open a DB file.");
	}
	write_header(fdata->dbfid, metadata);

#ifndef SD_STORAGE
	sprintf(name, DB_DIR "/" DATA_DIR "/" FFT_FILE, code, palette, pos, (int)ch + 1, metadata->nfile);
	fdata->fft_fid = async_file_open(name);
	if (fdata->fft_fid < 0) {
		perror("Failed to open an FFT file.");
	}
	write_header(fdata->fft_fid, metadata);
#endif
}

static void record_outlier(struct db_metadata *metadata, struct db_file_data *fdata)
{
	if (fdata->err_fp == NULL) {
		char name[256];
		sprintf(name, DB_DIR "/" DATA_DIR "/" ERROR_FILE,
			metadata->code, metadata->palette, metadata->pos, (int)metadata->channel + 1, metadata->nfile);
		fdata->err_fp = fopen(name, "w");
		if (fdata->err_fp == NULL) {
			perror("Failed to open an error file.");
			return;
		}
		fprintf(fdata->err_fp, "Found an outlier value.\n");
		fflush(fdata->err_fp);
	}
	if (outlier_fp != NULL &&
	    metadata->nfile < NUM_FILE_PER_DIR &&
	    metadata->outlier_count == 1) {
		fprintf(outlier_fp, "%02x_%02x_%02x_%05d\n",
			metadata->code, metadata->palette, metadata->pos, metadata->nfile);
		fflush(outlier_fp);
	}
}

/* Note: code, palette, and pos are for the data after the trigger.
 *       overflow is for the data before the trigger. */
static void trigger_handler(int code, int palette, int pos, int overflow)
{
	char name[256];
	int i;
	time_t start_time;

	fprintf(stderr, "Got trigger code=%d palette=%d position=%d\n", code, palette, pos);

	if (expfp != NULL) {
		fclose(expfp);
		expfp = NULL;
	}

	if (fft_expfp != NULL) {
		fclose(fft_expfp);
		fft_expfp = NULL;
	}

	if (overflow) {
		fprintf(stderr, "Detected overflow.\n");
	}

	start_time = time(NULL);
	for (i = 0; i < NUM_CHANNEL; ++i) {
		db_metadata_list[i].overflow = overflow;
		if (db_file_data_list[i].dbfid >= 0) {
			write_header(db_file_data_list[i].dbfid, &db_metadata_list[i]);
			async_file_close(db_file_data_list[i].dbfid);
			db_file_data_list[i].dbfid = -1;
		}
		if (db_file_data_list[i].fft_fid >= 0) {
			write_header(db_file_data_list[i].fft_fid, &db_metadata_list[i]);
			async_file_close(db_file_data_list[i].fft_fid);
			db_file_data_list[i].fft_fid = -1;
		}
		if (db_file_data_list[i].err_fp != NULL) {
			fclose(db_file_data_list[i].err_fp);
			db_file_data_list[i].err_fp = NULL;
		}
		if (db_metadata_list[i].written > 0) {
			write_trigger_summary(&db_metadata_list[i]);
		}

		db_metadata_list[i].written = 0;
		db_metadata_list[i].start_time = start_time;
		db_metadata_list[i].channel = i;
		db_metadata_list[i].code = (uint32_t)code;
		db_metadata_list[i].palette = (uint32_t)palette;
		db_metadata_list[i].pos = (uint32_t)pos;
		db_metadata_list[i].outlier_count = 0;
		db_metadata_list[i].fft_outlier_count = 0;
		db_metadata_list[i].status = 0;
	}

	load_nfile();

	for (i = 0; i < NUM_CHANNEL; ++i) {
		open_datafile(&db_metadata_list[i], &db_file_data_list[i]);
	}

	sprintf(name, DB_DIR "/" DATA_DIR "/" EXPECTED_FILE, code, palette, pos);
	expfp = fopen(name, "rb");
	if (expfp == NULL) {
		perror("Failed to open expected-value file.");
	}

	sprintf(name, DB_DIR "/" DATA_DIR "/" FFT_EXPECTED_FILE, code, palette, pos);
	fft_expfp = fopen(name, "rb");
	if (fft_expfp == NULL) {
		perror("Failed to open FFT expected-value file.");
	}
}

static void data_handler(const void *data, size_t length, const void *fft_data, size_t fft_length)
{
	uint16_t *channel_data[NUM_CHANNEL] = { 0 };
	int16_t *channel_fft_data[NUM_CHANNEL] = { 0 };
	static uint16_t expected[FIFO_SIZE / NUM_CHANNEL / sizeof(uint16_t) * 2];
	static uint16_t fft_expected[FIFO_SIZE / NUM_CHANNEL / sizeof(uint16_t) * 2];

	static int wcount = 0;
	size_t channel_length; /* in bytes  */
	size_t channel_fft_length; /* in bytes  */
	size_t rlen;
	int c;
	size_t i;

	const uint16_t *idata = (uint16_t *)data;
	const int16_t *ifft_data = (int16_t *)fft_data;

	if (length % NUM_CHANNEL != 0) {
		fprintf(stderr, "Length of data must be multiple of %d but is %zu.\n", NUM_CHANNEL, length);
	}
	channel_length = length / NUM_CHANNEL;
	channel_fft_length = fft_length / NUM_CHANNEL;

	/* TODO: Support 4ch separated-triggers. */
	if (HEADER_SIZE + db_metadata_list[0].written + channel_length > MAX_FILE_SIZE) {
		trigger_handler(
			db_metadata_list[0].code,
			db_metadata_list[0].palette,
			db_metadata_list[0].pos,
			0);
	}

	for (c = 0; c < NUM_CHANNEL; ++c) {
		channel_data[c] = async_file_malloc(FIFO_SIZE);
		if (channel_data[c] == NULL) {
			fprintf(stderr, "Allocation failed.\n");
			goto err;
		}
		channel_fft_data[c] = async_file_malloc(FIFO_SIZE * 2);
		if (channel_fft_data[c] == NULL) {
			fprintf(stderr, "Allocation failed.\n");
			goto err;
		}
		for (i = 0; i < length / sizeof(uint16_t) / NUM_CHANNEL; ++i) {
			size_t index = NUM_CHANNEL * i + NUM_CHANNEL - 1 - c;
			channel_data[c][i] = idata[index];
			channel_fft_data[c][i * 2] = ifft_data[index * 2];
			channel_fft_data[c][i * 2 + 1] = ifft_data[index * 2 + 1];
		}
	}

	if (length > FIFO_SIZE) {
		fprintf(stderr,
			"data_handler received too large data. (%u Bytes)",
			(unsigned int)length);
		length = FIFO_SIZE;
	}

	if (fft_expfp != NULL) {
		rlen = fread(fft_expected, 1, channel_fft_length, fft_expfp);
	} else {
		rlen = 0;
	}
	for (i = rlen / 2 / sizeof(uint16_t); i < channel_fft_length / sizeof(uint16_t) / 2; ++i) {
		fft_expected[i * 2]     = 0x0000;
		fft_expected[i * 2 + 1] = 0xffff;
	}
	for (c = 0; c < NUM_CHANNEL; ++c) {
		for (i = 0; i < rlen / 2 / sizeof(uint16_t); ++i) {
			int16_t real = channel_fft_data[c][2 * i];
			int16_t imag = channel_fft_data[c][2 * i + 1];
			int32_t actual = (int32_t)real * real + (int32_t)imag * imag;
			int32_t max = (int32_t)fft_expected[i * 2];
			int32_t min = (int32_t)fft_expected[i * 2 + 1];
			if ((max * max < actual ||
			     min * min > actual) &&
			    db_metadata_list[c].fft_outlier_count < UINT32_MAX) {
				++db_metadata_list[c].fft_outlier_count;
			}
		}
	}

	if (expfp != NULL) {
		rlen = fread(expected, 1, channel_length * 2, expfp);
	} else {
		rlen = 0;
	}
	for (i = rlen / 2 / sizeof(uint16_t); i < channel_length / sizeof(uint16_t); ++i) {
		expected[i * 2]     = 0x0000;
		expected[i * 2 + 1] = 0xffff;
	}
	for (c = 0; c < NUM_CHANNEL; ++c) {
		for (i = 0; i < rlen / 2 / sizeof(uint16_t); ++i) {
			if (expected[i * 2    ] < channel_data[c][i] ||
			    expected[i * 2 + 1] > channel_data[c][i]) {
				if (db_metadata_list[c].outlier_count < UINT32_MAX) {
					++db_metadata_list[c].outlier_count;
				}
				record_outlier(&db_metadata_list[c], &db_file_data_list[c]);
			}
		}
	}

	write_data_summary(stdout, channel_length, channel_data, channel_fft_data, expected, fft_expected);

	for (c = 0; c < NUM_CHANNEL; ++c) {
		if (db_file_data_list[c].dbfid >= 0) {
#ifdef SD_STORAGE
			async_file_write(db_file_data_list[c].dbfid, channel_data[c], channel_length);
#else
			if (wcount == FLUSH_COUNT / NUM_CHANNEL * c) {
				async_file_write_and_sync(db_file_data_list[c].dbfid, channel_data[c], channel_length);
			} else {
				async_file_write(db_file_data_list[c].dbfid, channel_data[c], channel_length);
			}
#endif
		} else {
			async_file_free(channel_data[c]);
		}

#ifdef SD_STORAGE
		async_file_free(channel_fft_data[c]);
#else
		if (db_file_data_list[c].fft_fid >= 0) {
			if (wcount == FLUSH_COUNT / NUM_CHANNEL * c) {
				async_file_write_and_sync(db_file_data_list[c].fft_fid, channel_fft_data[c], channel_fft_length);
			} else {
				async_file_write(db_file_data_list[c].fft_fid, channel_fft_data[c], channel_fft_length);
			}
		} else {
			async_file_free(channel_data[c]);
		}
#endif
		db_metadata_list[c].written += channel_length;
	}
	wcount = (wcount + 1) % FLUSH_COUNT;
	return;

err:
	for (c = 0; c < NUM_CHANNEL; ++c) {
		async_file_free(channel_data[c]);
		async_file_free(channel_fft_data[c]);
	}
}
#endif /* #ifdef PETALINUX */

int main(int argc, char** argv)
{
#ifdef PETALINUX
	init_summarizer();
	init_db();
	async_file_init();
	trigger_handler(DEFAULT_CODE, DEFAULT_PALETTE, DEFAULT_POS, 0);
	start_sensor(atoi(argv[3]), atoi(argv[4]), trigger_handler, data_handler);
	dispose_summarizer();
	async_file_finalize();
#else
    start_mic_sensor(argv[1]);
#endif

	return 0;
}
