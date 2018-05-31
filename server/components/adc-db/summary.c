#include "summary.h"

#define FFT_EXPECTED_LENGTH 4096
#define FFT_PERIOD 4

static uint16_t sqrt_u16(int32_t n)
{
	int32_t x = n / 2;
	int32_t next_x;

	if (n <= 1) {
		return n;
	}

	while (1) {
		next_x = (x + n / x) / 2;
		if (next_x >= x) {
			break;
		}
		x = next_x;
	}

	return (uint16_t)x;
}

static void calc_fft_norm(const int16_t *fft_buf, uint16_t *norm_buf, size_t len)
{
	size_t i;
	int16_t real, imag;
	for (i = 0; i < len; i++) {
		real = (int32_t)fft_buf[i * 2];
		imag = (int32_t)fft_buf[i * 2 + 1];
		norm_buf[i] = sqrt_u16(
			(int32_t)real * real + (int32_t)imag * imag);
	}
}

void write_data_summary(FILE* fp, int r,
	uint16_t *read_buf[],
	int16_t *fft_buf[],
	const uint16_t *expected,
	const uint16_t *fft_expected)
{
	static uint32_t acc_count = 0;
	static unsigned int fft_count = 0;
	static uint16_t max_data[NUM_CHANNEL] = { 0, 0, 0, 0 };
	static uint16_t min_data[NUM_CHANNEL] = { UINT16_MAX, UINT16_MAX, UINT16_MAX, UINT16_MAX };
	static uint16_t max_expected = 0, min_expected = 0xffff;
	const size_t n = r / sizeof(**read_buf);

	int c;
	size_t i;
	uint16_t value;

	for (i = 0; i < n; ++i) {
		for (c = 0; c < NUM_CHANNEL; ++c) {
			value = read_buf[c][i];
			if (max_data[c] < value) {
				max_data[c] = value;
			}
			if (min_data[c] > value) {
				min_data[c] = value;
			}
		}

		value = expected[i * 2];
		if (max_expected < value) {
			max_expected = value;
		}

		value = expected[i * 2 + 1];
		if (min_expected > value) {
			min_expected = value;
		}

#ifdef DEBUG_BIN
		fprintf(stderr, "Get %d/%d/%d\n", (unsigned int)read_buf[0][i], (unsigned int)expected[i * 2], (unsigned int)expected[i * 2 + 1]);
#endif
		// fprintf(stderr, "%d/%d\n", expected[i * 2], expected[i * 2 + 1]);

		++acc_count;
		if (acc_count == FIFO_SIZE) {
			static const uint16_t header[] = { 0, 8 * NUM_CHANNEL, 0 };
			uint16_t out_buf[4 * NUM_CHANNEL];
			for (c = 0; c < NUM_CHANNEL; ++c) {
				out_buf[c * 4]     = max_data[c];
				out_buf[c * 4 + 1] = min_data[c];
				out_buf[c * 4 + 2] = max_expected;
				out_buf[c * 4 + 3] = min_expected;
				max_data[c] = 0;
				min_data[c] = UINT16_MAX;
			}
			max_expected = 0;
			min_expected = UINT16_MAX;
			// fprintf(stderr, "sizeof(out_buf): %d\n", sizeof(out_buf));
#ifdef DEBUG_BIN
			fprintf(stderr, "Write %d/%d/%d/%d\n", (unsigned int)out_buf[0], (unsigned int)out_buf[1], (unsigned int)out_buf[2], (unsigned int)out_buf[3]);
#endif
			fwrite(header, sizeof(header), 1, fp);
			fwrite(out_buf, sizeof(out_buf), 1, fp);
			acc_count = 0;

			if (fft_count == 0) {
				for (c = 0; c < NUM_CHANNEL; ++c) {
					static uint16_t out_buf[FIFO_SIZE / sizeof(uint16_t) / NUM_CHANNEL];
					uint16_t header[] = { 0x01, 2 + r, 0, c };
					calc_fft_norm(fft_buf[c], out_buf, n);
					fwrite(header, sizeof(header), 1, fp);
					fwrite(out_buf, r, 1, fp);
				}
				uint16_t header[] = { 0x02, FFT_EXPECTED_LENGTH, 0 };
				fwrite(header, sizeof(header), 1, fp);
				fwrite(fft_expected, FFT_EXPECTED_LENGTH, 1, fp);
			}
			fflush(fp);
			fft_count = (fft_count + 1) % FFT_PERIOD;
		}
	}
}

#include <string.h>
#include <time.h>
#include <pthread.h>
#include <unistd.h>
#include "MQTTClient.h"
#define ADDRESS     "tcp://192.168.0.20:1883"
#define CLIENTID    "Olive-ADC"
#define QOS         0
#define TOPIC       "Olive ADC"
#define CONNECTION_CHALLENGE_INTERVAL 2
#define TIMEOUT     1000ul

static volatile int mqtt_enabled = 0;
static MQTTClient client;
static MQTTClient_connectOptions conn_opts = MQTTClient_connectOptions_initializer;
static pthread_t mqtt_connection_thread;

void write_trigger_summary(const struct db_metadata *metadata)
{
	MQTTClient_message pubmsg = MQTTClient_message_initializer;
	char payload[256];

	char time_str[64];
	struct tm tm_time;

	time_t start_time = metadata->start_time + 9 * 60 * 60;

	if (!mqtt_enabled) {
		return;
	}

#ifdef _POSIX_C_SOURCE
	gmtime_r(&start_time, &tm_time);
#else
       struct tm *pm = &tm_time;
       pm = localtime(&start_time);
#endif
	strftime(time_str, sizeof(time_str), "%FT%H:%M:%S+09:00", &tm_time);
	time_str[sizeof(time_str) - 1] = '\0';

	pubmsg.payloadlen = sprintf(payload,
		"{\"code\":%d,"
		"\"palette\":%d,"
		"\"position\":%d,"
		"\"time\":\"%s\","
		"\"raw_error_count\":%d,"
		"\"fft_error_count\":%d,"
		"\"num_samples\":%lu,"
		"\"frequency\":250000,"
		"\"overflow\":%s,"
		"\"status\":%d}",
		(int)metadata->code,
		(int)metadata->palette,
		(int)metadata->pos,
		time_str,
		metadata->outlier_count,
		metadata->fft_outlier_count,
		(unsigned long)metadata->written / 2,
		metadata->overflow ? "true" : "false",
		(int)metadata->status
		);
	pubmsg.payload = payload;
	pubmsg.qos = QOS;
	pubmsg.retained = 0;

	MQTTClient_publishMessage(client, TOPIC, &pubmsg, NULL);
}

static void *connection_thread(void *args)
{
	int rc;

	do {
		rc = MQTTClient_connect(client, &conn_opts);
		sleep(CONNECTION_CHALLENGE_INTERVAL);
	} while (rc != MQTTCLIENT_SUCCESS);

	mqtt_enabled = 1;

	return NULL;
}

void init_summarizer(void)
{
	conn_opts.keepAliveInterval = 20;
	conn_opts.cleansession = 1;

	MQTTClient_create(&client, ADDRESS, CLIENTID,
		MQTTCLIENT_PERSISTENCE_NONE, NULL);

	pthread_create(&mqtt_connection_thread, NULL, &connection_thread, NULL);
}

void dispose_summarizer(void)
{
	MQTTClient_disconnect(client, TIMEOUT);
	MQTTClient_destroy(&client);
}
