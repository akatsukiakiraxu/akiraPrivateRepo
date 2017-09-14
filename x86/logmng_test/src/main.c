/*
 * main.c
 *
 *  Created on: 2017/09/05
 *  Copyright: verifore.jp
 *  Author: xu
 */

#include <stdio.h>
#include <stdlib.h>
#include <io.h>
#include <termios.h>    // terminal I/F
#include <sys/types.h>  // open / select
#include <sys/stat.h>   // open / select
#include <fcntl.h>      // open / select
#include <string.h>

#include "logmng.h"

int logmng_decode(int fd);

/**
 * @brief
 * シリアルポートを初期化する
 * @par 詳細仕様
 * - シリアルポートを初期化する
 * - 初期化に成功した場合、デバイスファイルディスクリプタを返却する
 * @param [in] device デバイス名
 * @retval fd デバイスファイルディスクリプタ
 *
 */
static int init_serial(char* device) {
    int fd;
    struct termios tty;

    if ((fd = open(device, O_RDWR | O_NOCTTY)) == -1) {
        fprintf(stderr, "can't open %s.\n", device);
        return -1;
    }
    memset(&tty, 0, sizeof(tty));
    tty.c_cflag = CS8 | CLOCAL | CREAD;
    tty.c_cc[VMIN] = 1;
    tty.c_cc[VTIME] = 0;
    cfsetospeed(&tty, B115200);
    cfsetispeed(&tty, B115200);
    tcflush(fd, TCIFLUSH);
    tcsetattr(fd, TCSANOW, &tty);

    return fd;
}


/**
 * @brief
 * メイン関数
 * @par 詳細仕様
 * - シリアルポートを初期化する
 * - ログの取得・変換・出力を行う
 * @param [in] argc プログラムオプション数
 * @param [in] argv プログラムオプション配列
 * @retval 0 常に0
 *
 */
int main(int argc, char** argv) {
    char* fname;
    int res, fd;
    // ファイルからバイナリに変換
    fname = argv[1];
    fd = init_serial(fname);
    res = logmng_decode(fd);
    if (res != 0) {
        fprintf(stdout, "logmng_decode error. %X\n\n", res);
        return res;
    }
    fprintf(stderr, "logmng_decode sucess.\n");

    return 0;
}
