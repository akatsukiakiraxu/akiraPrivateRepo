/*
 * logmng.c
 *
 *  Created on: 2017/09/05
 *  Copyright: verifore.jp
 *  Author: xu
 */

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <io.h>
#include <unistd.h>     // read / write / close
#include <logmng.h>

#define STDOUT(...) fprintf(stdout, __VA_ARGS__)
#define STDERR(...) fprintf(stderr, __VA_ARGS__)
#define MODE_UNSPEC (0)
#define MODE_ASCII  (1)
#define MODE_BINARY (2)
#define MODE_LINE   (3)
#define LOG_LEN (16)
#define BUFF_SIZE (1024)
/**
 * @attetion
 * ここの文字列定義は必ずSCM側の定義に合わせること！
 *
 */
static LOGMNG_FID_T fidConvertTable[] = {
      { FID_UNUSE    , "------"     , }
    , { FID_SND      , "SND   "     , }
    , { FID_ASIB     , "ASIB  "     , }
    , { FID_ROMIF    , "ROMIF "     , }
    , { FID_ICM      , "ICM   "     , }
    , { FID_RDC      , "RDC   "     , }
    , { FID_DMAC     , "DMAC  "     , }
    , { FID_SSC0     , "SSC0  "     , }
    , { FID_SSC1     , "SSC1  "     , }
    , { FID_SSC2     , "SSC2  "     , }
    , { FID_SSC3     , "SSC3  "     , }
    , { FID_UART0    , "UART0 "     , }
    , { FID_UART1    , "UART1 "     , }
    , { FID_UART2    , "UART2 "     , }
    , { FID_UART3    , "UART3 "     , }
    , { FID_GPIO_0   , "GPIO0 "     , }
    , { FID_GPIO_1   , "GPIO1 "     , }
    , { FID_GPIO_2   , "GPIO2 "     , }
    , { FID_I2C0     , "I2C0  "     , }
    , { FID_I2C1     , "I2C1  "     , }
    , { FID_TIMER0   , "TIMER0"     , }
    , { FID_TIMER1   , "TIMER1"     , }
    , { FID_USER0    , "USER0 "     , }
    , { FID_USER1    , "USER1 "     , }
    , { FID_USER2    , "USER2 "     , }
    , { FID_LOG      , "LOG   "     , }
    , { FID_LOGIRQ   , "LOGIRQ"     , }
    , { FID_DEBUG    , "DEBUG "     , }
    , { _fid_tail    , "='w'= "     , }
};

static LOGMNG_MID_T midConvertTable[] = {
      { MID_OK              ,  "Success"               , }    // 成功
    , { MID_ERR             ,  "Error"                 , }    // エラー
    , { MID_SPEC_ERR        ,  "Performance Error"     , }    // 性能エラー
    , { MID_INTR_START      ,  "Interrupt Start"       , }    // 割込み処理開始
    , { MID_INTR_END        ,  "Interrupt End"         , }    // 割込み処理完了
    , { MID_INTR_ERR        ,  "Interrupt Error"       , }    // 割込み処理エラー
    , { MID_MSG_SEND_START  ,  "Message Send Start"    , }    // 外部機能へのメッセージ送信開始
    , { MID_MSG_SEND_END    ,  "Message Send End"      , }    // 外部機能へのメッセージ送信完了
    , { MID_MSG_SEND_ERR    ,  "Message Send Error"    , }    // 外部機能へのメッセージ送信エラー
    , { MID_MSG_RECV_START  ,  "Message Receive Start" , }    // 外部機能からのメッセージ受信開始
    , { MID_MSG_RECV_END    ,  "Message Receive End"   , }    // 外部機能からのメッセージ受信完了
    , { MID_MSG_RECB_ERR    ,  "Message Receive Error" , }    // 外部機能からのメッセージ受信エラー
    , { MID_TEST_START      ,  "Test Start"            , }    // 検証プログラム開始
    , { MID_TEST_END        ,  "Test End"              , }    // 検証プログラム完了
    , { MID_TEST_ERR        ,  "Test Error"            , }    // 検証プログラムエラー
    , { MID_CSUM_START      ,  "Checksum Start"        , }    // チェックサム計算開始
    , { MID_CSUM_END        ,  "Checksum End"          , }    // チェックサム計算終了
    , { MID_CSUM_ERR        ,  "Checksum Error"        , }    // チェックサムエラー
    , { MID_DATA_SIZE       ,  "Data size"             , }    // データサイズ
    , { MID_DATA_ERR        ,  "Data Error"            , }    // データエラー
    , { MID_DATA_TX_START   ,  "Data Tx Start"         , }    // データ送信開始
    , { MID_DATA_TX_END     ,  "Data Tx End"           , }    // データ送信完了
    , { MID_DATA_TX_ERR     ,  "Data Tx Error"         , }    // データ送信エラー
    , { MID_DATA_RX_START   ,  "Data Rx Start"         , }    // データ受信開始
    , { MID_DATA_RX_END     ,  "Data Rx End"           , }    // データ受信完了
    , { MID_DATA_RX_ERR     ,  "Data Rx Error"         , }    // データ受信エラー
    , { MID_DATA_TIMEOUT    ,  "Data Timeout"          , }    // データタイムアウト
    , { MID_TIMEOUT         ,  "Timeout"               , }    // タイムアウト
    , { MID_DELAY_ERR       ,  "Delay Error"           , }    // 遅延エラー
    , { MID_EVENT_START     ,  "Event Start"           , }    // 開始イベント
    , { MID_EVENT_END       ,  "Event End"             , }    // 終了イベント
    , { _mid_tail           ,  "#undefined message# "  , }
};

/**
 * @brief
 * ファンクションIDを文字列に変換する
 * @par 詳細仕様
 * - 指定されたFIDに対応する文字列をfidConvertTableから取得する
 * @param [in] FID ファンクションID
 * @retval ファンクションIDの文字列
 *
 */
static char* logmng_fid2str(LOGMNG_FID FID) {
    LOGMNG_FID_T* p = fidConvertTable;
    while (p->FID != _fid_tail) {
        if (p->FID == FID)
            break;
        p++;
    }
    return p->STR;
}

/**
 * @brief
 * メッセージIDを文字列に変換する
 * @par 詳細仕様
 * - 指定されたMIDに対応する文字列をmidConvertTableから取得する
 * @param [in] MID メッセージID
 * @retval メッセージIDの文字列
 *
 */
static char* logmng_mid2str(LOGMNG_MID MID) {
    LOGMNG_MID_T* p = midConvertTable;
    while (p->MID != _mid_tail) {
        if (p->MID == MID)
            break;
        p++;
    }
    return p->STR;
}


/**
 * @brief
 * SCMから送られてきたログデータを取得し、視認用の文字列に変換して出力する
 * @par 詳細仕様
 * - 渡されたデバイスファイルディスクリプタをポーリングする
 * - データが送られてきた場合、開始コードをチェックする
 *   -# 開始コードが[BOL]の場合、ログデータと認識して変換して出力する
 *   -# それ以外の場合、通常ログと認識して変換せずにそのまま出力する
 * @param [in] fd シリアルポートのデバイスファイルディスクリプタ
 * @retval 0 処理成功
 * @retval -1 処理失敗
 *
 */
int logmng_decode(int fd) {
    uint8_t mode                = MODE_UNSPEC;
    uint8_t read_buf[BUFF_SIZE] = {0};
    int read_idx                = 0;
    uint8_t tmp_buf[BUFF_SIZE]  = {0};
    int tmp_idx                 = 0;
    int fd_key = fileno(stdin);

    fprintf(stdout, "logmng_decode start\n\r");

    while (1) {
        fd_set fdset;
        FD_ZERO(&fdset);
        FD_SET(fd_key, &fdset);
        FD_SET(fd, &fdset);

        int ret = select(fd + 1, &fdset, NULL, NULL, NULL);
        if (ret < 0) {
            STDERR("select() interrupts");
        }
        // read from stdin
        if (FD_ISSET(fd_key, &fdset)) {
            int ret = read(fd_key, read_buf, sizeof(read_buf) - 1);
            if (ret < 0) {
                STDERR("read error\n");
                goto error;
            }
            read_buf[ret] = 0;
            write(fd, read_buf, ret);
        }
        // read from target
        if (FD_ISSET(fd, &fdset)) {
            int ret = read(fd, read_buf, sizeof(read_buf) - 1);
            if (ret < 0) {
                STDERR("read error\n");
                goto error;
            }
            read_buf[ret] = 0;
            for (read_idx = 0 ; read_idx < ret ; read_idx++) {
                // 共通処理
                tmp_buf[tmp_idx++] = read_buf[read_idx];
                if (read_buf[read_idx] == '\n') {
                    // ライン出力モード開始を検知
                    if (memcmp((char*)tmp_buf, STR_BOL, sizeof(STR_BOL)) == 0) {
                        mode = MODE_LINE;
                        memset(tmp_buf, 0, sizeof(tmp_buf));
                        tmp_idx = 0;
                        continue;
                    }
                    // ライン出力モード終了を検知
                    if (memcmp((char*)tmp_buf, STR_EOL, sizeof(STR_EOL)) == 0) {
                        mode = MODE_UNSPEC;
                        memset(tmp_buf, 0, sizeof(tmp_buf));
                        tmp_idx = 0;
                        continue;
                    }
                }
                // モード別処理
                switch (mode) {
                default:
                    STDERR("mode = %u error", mode);
                    goto error;
                case MODE_UNSPEC:
                    // 改行コード毎に標準出力
                    if (read_buf[read_idx] == '\n') {
                        STDOUT("%s", tmp_buf);
                        memset(tmp_buf, 0, sizeof(tmp_buf));
                        tmp_idx = 0;
                        break;
                    }
                    break;
                case MODE_LINE:
                    // 16Byte毎に出力
                    if (tmp_idx == LOG_LEN) {
                        LOGMNG_Msg_t* pMsg = (LOGMNG_Msg_t*)tmp_buf;
                        char split = ',';
                        STDOUT("0x%08X%c%1u%c%6s%c%-24s%c0x%08X%c0x%08X\n",
                               pMsg->SYSMNTR, split,
                               pMsg->LV, split,
                               logmng_fid2str(pMsg->FID), split,
                               logmng_mid2str(pMsg->MID), split,
                               pMsg->ARG0, split,
                               pMsg->ARG1);
                        memset(tmp_buf, 0, sizeof(tmp_buf));
                        tmp_idx = 0;
                        break;
                    }
                    break;
                }
            }
            fflush(stdout);
        }
    }
    close(fd);
    close(fd_key);
    return 0;

error:
    close(fd);
    close(fd_key);
    return -1;
}

