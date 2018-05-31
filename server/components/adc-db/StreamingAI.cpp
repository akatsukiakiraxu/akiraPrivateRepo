/*******************************************************************************
Copyright (c) 2018 Fixstars Co., Ltd.
********************************************************************************

================================================================================
REVISION HISTORY


================================================================================
--------------------------------------------------------------------------------
$Log:  $
--------------------------------------------------------------------------------
$NoKeywords:  $
*/
/******************************************************************************
*
* File name:
*    StreamingAI.cpp
*
* Description:
*
******************************************************************************/
#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <vector>
#include <semaphore.h>
#include <time.h>
#include <string>
#include <unistd.h>
#include <thread>
#include "Singleton.hpp"
#include "AsyncControlImpl.hpp"
#include "PropertyUtility.hpp"
#include "AsyncCommand.hpp"
#include "StreamingAI.h"
#include "debug.h"
#include "bdaqctrl.h"
#include "ChangeSettingUtility.hpp"

using namespace Automation::BDaq;
using namespace std;
//-----------------------------------------------------------------------------------
// define some global variables
//-----------------------------------------------------------------------------------
sem_t sem_msg;
sem_t sem_cmd;
int channelCountMax = 0;
struct timespec tsPre = {0, 0};
const int nano2sec = 1000 * 1000 * 1000;
bool isShowDebugMsg = false;
FILE* fpw = NULL;
PropertyUtility& m_propertyUtility = Singleton<PropertyUtility>::GetInstance();
AsyncControlImpl& m_pAsyncControlImpl = Singleton<AsyncControlImpl>::GetInstance();
double userDataBuffer[1024*1024] = {0};
InstantDiCtrl* instantDiCtrl =  NULL;
bool isChangedSetting = false;
// This class is used to deal with 'DataReady' Event, we should overwrite the virtual function BfdAiEvent.
class DataReadyHandler : public BfdAiEventListener {
  public:
    virtual void BDAQCALL BfdAiEvent(void* sender, BfdAiEventArgs* args) {
        ADCDB_DEBUG(isShowDebugMsg, "start");
        BufferedAiCtrl* bufferedAiCtrl = (BufferedAiCtrl*)sender;
        int chanCount = bufferedAiCtrl->getScanChannel()->getChannelCount();
        int32 dataCount = args->Count;
        if (chanCount == 0 || chanCount > m_propertyUtility.getDeviceInfomation().SampleChannelCount || bufferedAiCtrl == NULL || dataCount <= 0) {
            ADCDB_ERR("Error chanCount[%d] SampleChannelCount[%d] bufferedAiCtrl[%p] dataCount[%d] ",
                      chanCount, m_propertyUtility.getDeviceInfomation().SampleChannelCount, bufferedAiCtrl, dataCount);
            return;
        }
        uint32_t bufferSize = dataCount * sizeof(double);
        bufferedAiCtrl->GetData(dataCount, userDataBuffer);
        ADCDB_DEBUG(isShowDebugMsg, "dataCount %d", dataCount);

        dataInfo_t dataInfo;
        dataInfo.chanStart = bufferedAiCtrl->getScanChannel()->getChannelStart();
        dataInfo.chanCount = chanCount;
        dataInfo.dataSize = bufferSize;
        dataInfo.pDataBuf = &userDataBuffer[0];

        AsyncCommand* pCmd = new AsyncCommand();
        pCmd->setDataInfo(&dataInfo);
        m_pAsyncControlImpl.addCmd(pCmd);
        ADCDB_DEBUG(isShowDebugMsg, "addCmd[%p]", pCmd);
        sem_post(&sem_cmd);
    }
};

// This class is used to deal with 'Overrun' Event, we should overwrite the virtual function BfdAiEvent.
class OverrunHandler : public BfdAiEventListener {
  public:
    virtual void BDAQCALL BfdAiEvent(void* sender, BfdAiEventArgs* args) {
        ADCDB_ERR("Streaming AI Overrun: offset = %d, count = %d", args->Offset, args->Count);
    }
};

// This class is used to deal with 'CacheOverflow' Event, we should overwrite the virtual function BfdAiEvent.
class CacheOverflowHandler : public BfdAiEventListener {
  public:
    virtual void BDAQCALL BfdAiEvent(void* sender, BfdAiEventArgs* args) {
        ADCDB_ERR(" Streaming AI Cache Overflow: offset = %d, count = %d", args->Offset, args->Count);
    }
};

// This class is used to deal with 'Stopped' Event, we should overwrite the virtual function BfdAiEvent.
class StoppedHandler : public BfdAiEventListener {
  public:
    virtual void BDAQCALL BfdAiEvent(void* sender, BfdAiEventArgs* args) {
        ADCDB_ERR("Streaming AI stopped: offset = %d, count = %d", args->Offset, args->Count);
    }
};




extern "C" {

    void cmdExeThread() {
        uint16_t triggerFlagPre = 0;
        uint16_t tiggerFlag = 0;
        int triggerStatus = TS_UNKNOWN;
        uint64_t sendedPoints = 0, timeoutPoints = 0;
        while (true) {
            sem_wait(&sem_cmd);
            AsyncCommand* cmd = m_pAsyncControlImpl.getCmd();
            if (cmd != NULL) {
                tiggerFlag = 0;
                /*
                * read trigger
                */
                if (m_propertyUtility.getSoftCtrl().SoftTriggerType == 1) {
                    struct timespec tsNow;
                    clock_gettime(CLOCK_REALTIME, &tsNow);
                    time_t pastTime_ns = (tsNow.tv_sec - tsPre.tv_sec) * nano2sec + (tsNow.tv_nsec - tsPre.tv_nsec);
                    ADCDB_DEBUG(isShowDebugMsg, "past time : %10ld", pastTime_ns);
                    time_t softTriggerInterval_ns = (time_t)m_propertyUtility.getSoftCtrl().SoftTrigger.IntervalMs * 1000 * 1000;
                    if (pastTime_ns >= softTriggerInterval_ns) {
                        tiggerFlag = 1;
                        tsPre.tv_sec = tsNow.tv_sec;
                        tsPre.tv_nsec = tsNow.tv_nsec;
                    }
                } else
                    if (m_propertyUtility.getSoftCtrl().SoftTriggerType == 0) {
                        uint8  bufferForReadingTrigger[64] = {0};
                        instantDiCtrl->Read(0, 1, bufferForReadingTrigger);
                        uint16 triggerFlagNow = (bufferForReadingTrigger[0] & 0x1);
                        if ((triggerFlagPre == 0) && (triggerFlagNow == 1)) {
                            tiggerFlag = 1;
                        }
                        triggerFlagPre = triggerFlagNow;
                    } else {
                        /* do nothing */
                    }
                ADCDB_DEBUG(isShowDebugMsg, "getCmd[%p]", cmd);
                cmd->setTriggerStatus(triggerStatus);
                cmd->setTrigger(tiggerFlag);
                cmd->setSendedPoints(sendedPoints);
                cmd->setPointsAfterLastTrigger(timeoutPoints);
                cmd->execute();
                triggerStatus = cmd->getTiggerStatus();
                sendedPoints = cmd->getSendedPoints();
                timeoutPoints = cmd->getPointsAfterLastTrigger();
                if (m_propertyUtility.getSoftCtrl().IsDumpData && fpw != NULL) {
                    fwrite(cmd->getDataInfo()->pDataBuf, 1, cmd->getDataInfo()->dataSize, fpw);
                }
                delete cmd;
                cmd = NULL;
            }
        }
    }

    void changeSettingThread() {
        for (string filename; getline(cin, filename);) {
            ChangeSettingUtility *pChangeSettingUtility = new ChangeSettingUtility();
            pChangeSettingUtility->setSettingFileName(filename);
            if (pChangeSettingUtility->isExistSettingFile()) {
                pChangeSettingUtility->loadSetting();
                m_propertyUtility.notifyChangedSetting(pChangeSettingUtility);
                m_propertyUtility.saveProperties();
                isChangedSetting = true;
                if (!pChangeSettingUtility->isHoldSettingFile()) {
                    remove(filename.c_str());
                } else {
                    pChangeSettingUtility->dumpSetting();
                }
                sem_post(&sem_msg);
            }
            else {
                ADCDB_ERR("setting file not found!");
            }
            delete pChangeSettingUtility;
        }
    }


    void start_mic_sensor(char* propertyFileName) {
        fprintf(stderr, "start_mic_sensor\n");
        if (propertyFileName == NULL) {
            fprintf(stderr, "PropertyFileName not specified!\n");
            return;
        } else {
            m_propertyUtility.setPropertyFileName(propertyFileName);
            if (!m_propertyUtility.isExistPropertyFile()) {
                fprintf(stderr, "[%s] not found!\n", propertyFileName);
                return;
            }
        }
        ADCDB_DEBUG(isShowDebugMsg, "GetProperty success");
        m_propertyUtility.getProperties();
        isShowDebugMsg = m_propertyUtility.getSoftCtrl().IsShowDebugLog;

        ErrorCode ret = Success;
        sem_init(&sem_msg, 0, 0);
        sem_init(&sem_cmd, 0, 0);
        // Step 1: Create a 'BufferedAiCtrl' for buffered AI function.
        BufferedAiCtrl* bfdAiCtrl = AdxBufferedAiCtrlCreate();
        instantDiCtrl = AdxInstantDiCtrlCreate();

        // Step 2: Set the notification event Handler by which we can known the state of operation effectively.
        DataReadyHandler onDataReady;
        OverrunHandler onOverrun;
        CacheOverflowHandler onCacheOverflow;
        StoppedHandler onStopped;
        bfdAiCtrl->addDataReadyListener(onDataReady);
        bfdAiCtrl->addOverrunListener(onOverrun);
        bfdAiCtrl->addCacheOverflowListener(onCacheOverflow);
        bfdAiCtrl->addStoppedListener(onStopped);
        do {
            // Step 3: Select a device by device number or device description and specify the access mode.
            // in this example we use AccessWriteWithReset(default) mode so that we can
            // fully control the device, including configuring, sampling, etc.
            DeviceInformation devInfo(m_propertyUtility.getDeviceInfomation().Description.c_str());
            ret = bfdAiCtrl->setSelectedDevice(devInfo);
            ret = instantDiCtrl->setSelectedDevice(devInfo);
            ADCDB_DEBUG(isShowDebugMsg, "device open success");

            channelCountMax = bfdAiCtrl->getFeatures()->getChannelCountMax();
            if (channelCountMax > m_propertyUtility.getDeviceInfomation().ChannelCountMax) {
                ADCDB_ERR("channelCountMax[%d] is larger than %d which was read from property file!", channelCountMax, m_propertyUtility.getDeviceInfomation().ChannelCountMax);
                CHK_RESULT(-1);
            }
            //ICollection<ValueRange>* ranges = bfdAiCtrl->getFeatures()->getValueRanges();

            ConvertClock* pConvertClock = bfdAiCtrl->getConvertClock();
            pConvertClock->setRate(m_propertyUtility.getDeviceInfomation().SamplingRate);
            // Step 4: Set necessary parameters for Buffered AI operation,
            // Note: some of operation of this step is optional(you can do these settings via "Device Configuration" dialog).
            ScanChannel* scanChannel = bfdAiCtrl->getScanChannel();
            ret = scanChannel->setChannelStart(m_propertyUtility.getDeviceInfomation().StartChannel);
            CHK_RESULT(ret);
            ret = scanChannel->setChannelCount(m_propertyUtility.getDeviceInfomation().SampleChannelCount);
            CHK_RESULT(ret);
            ret = scanChannel->setIntervalCount(m_propertyUtility.getDeviceInfomation().IntervalCount);
            CHK_RESULT(ret);
            int32 sampleBufferSize = m_propertyUtility.getDeviceInfomation().SampleChannelCount * m_propertyUtility.getDeviceInfomation().IntervalCount;
            ret = scanChannel->setSamples(sampleBufferSize);
            CHK_RESULT(ret);
            ret = bfdAiCtrl->setStreaming(true);// specify the running mode: streaming buffered.
            CHK_RESULT(ret);

            AiChannelCollection* channels = bfdAiCtrl->getChannels();
            for (int i = 0; i < scanChannel->getChannelCount(); ++i) {
                channels->getItem(scanChannel->getChannelStart()+i).setValueRange(static_cast<ValueRange>(m_propertyUtility.getDeviceInfomation().ValueRange));
                channels->getItem(scanChannel->getChannelStart()+i).setSignalType(static_cast<AiSignalType>(m_propertyUtility.getDeviceInfomation().SignalType));
            }

            // Step 5: Prepare the buffered AI.
            ret = bfdAiCtrl->Prepare();
            CHK_RESULT(ret);
            ADCDB_DEBUG(isShowDebugMsg, "device Prepare success");

            // Step 6: Start buffered AI, the method will return immediately after the operation has been started.
            // We can get samples via event handlers.
            if (m_propertyUtility.getSoftCtrl().SoftTriggerType == 1) {
                clock_gettime(CLOCK_REALTIME, &tsPre);
                ADCDB_DEBUG(isShowDebugMsg, "%10ld.%9ld", tsPre.tv_sec, tsPre.tv_nsec);
            }
            if (m_propertyUtility.getSoftCtrl().IsDumpData) {
                fpw = fopen(m_propertyUtility.getSoftCtrl().DumpDataPath.c_str(), "ab");
            }

            ret = bfdAiCtrl->Start();
            ADCDB_DEBUG(isShowDebugMsg, "Start ret[%d]", ret);
            CHK_RESULT(ret);
            // Step 7: Do anything you are interesting while the device is acquiring data.
            auto cmdExeTh = std::thread([] {cmdExeThread();});
            auto changeSettingTh = std::thread([] {changeSettingThread();});
            ADCDB_DEBUG(isShowDebugMsg, "cmdExeThread start");

            while (1) {
                sem_wait(&sem_msg);
                if (isChangedSetting) {
                    ret = bfdAiCtrl->Stop();
                    CHK_RESULT(ret);
                    pConvertClock->setRate(m_propertyUtility.getDeviceInfomation().SamplingRate);
                    for (int i = 0; i < scanChannel->getChannelCount(); ++i) {
                        channels->getItem(scanChannel->getChannelStart()+i).setValueRange(static_cast<ValueRange>(m_propertyUtility.getDeviceInfomation().ValueRange));
                        channels->getItem(scanChannel->getChannelStart()+i).setSignalType(static_cast<AiSignalType>(m_propertyUtility.getDeviceInfomation().SignalType));
                    }
                    ret = bfdAiCtrl->Start();
                    CHK_RESULT(ret);
                    isChangedSetting = false;
                }
            }
            // step 8: Stop the operation if it is running.
            ret = bfdAiCtrl->Stop();
            CHK_RESULT(ret);
            cmdExeTh.join();
            changeSettingTh.join();
        } while (false);
        ADCDB_DEBUG(isShowDebugMsg, "Stop");
        // Step 9: Close device, release any allocated resource.
        bfdAiCtrl->Dispose();
        if (fpw) {
            fclose(fpw);
        }
        // If something wrong in this execution, print the error code on screen for tracking.
        if (BioFailed(ret)) {
            ADCDB_ERR("BioFailed[0x%08X]", ret);
        }
        return;
    }
} /* extern "C"*/
