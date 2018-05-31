#ifndef DEFINES_H
#define DEFINES_H
/* IOCTL defines */

/* IOCTL defines */
#define AXI_XADC_IOCTL_BASE                     'W'
#define AXI_XADC_GET_NUM_DEVICES                _IO(AXI_XADC_IOCTL_BASE, 0)
#define AXI_XADC_GET_DEV_INFO                   _IO(AXI_XADC_IOCTL_BASE, 1)
#define AXI_XADC_DMA_CONFIG                     _IO(AXI_XADC_IOCTL_BASE, 2)
#define AXI_XADC_DMA_START                      _IO(AXI_XADC_IOCTL_BASE, 3)
#define AXI_XADC_DMA_STOP                       _IO(AXI_XADC_IOCTL_BASE, 4)
#define AXI_AD7476_CONFIG                       _IO(AXI_XADC_IOCTL_BASE, 5)
#define AXI_XADC_DMA_RESET                      _IO(AXI_XADC_IOCTL_BASE, 6)
#define ADC_THRESHOLD_SETUP0                    _IO(AXI_XADC_IOCTL_BASE, 7)
#define ADC_THRESHOLD_SETUP1                    _IO(AXI_XADC_IOCTL_BASE, 8)
#define ADC_TRIGGER                             _IO(AXI_XADC_IOCTL_BASE, 9)
#define ADC_RESET                               _IO(AXI_XADC_IOCTL_BASE, 10)


#define FIFO_SIZE ((8*1024))
#define CONNECTION_EST_REG  0x00
#define RAW_DATA_REG        0x01        //- Raw data collection
#define XST_FAILURE -1
#define XST_SUCCESS  0
#define UART_BUF_SIZE 20

#define BAUDRATE B115200
#define DEV_NODE "/dev/ttyPS0"
#define _POSIX_SOURCE 1 /* POSIX compliant source */

#endif
