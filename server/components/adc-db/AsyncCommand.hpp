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
*    AsyncCommad.hpp
*
* Description:
*
******************************************************************************/
#ifndef _ASYNCCOMMAND_HPP_
#define _ASYNCCOMMAND_HPP_

#include <stdlib.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#include "PropertyUtility.hpp"

typedef struct dataInfo {
    uint32_t chanStart;
    uint32_t chanCount;
    uint32_t dataSize;
    double* pDataBuf;
} dataInfo_t;

enum {
    TS_UNKNOWN = -99,
    TS_LOW = 0,
    TS_WAIT_LOW,
    TS_WAIT_HIGH,
    TS_HIGH,
};

enum {
    EDGE_NOTFOUND = -1,
    EDGE_FALLINGDOWN = 0,
    EDGE_RASINGUP = 1,
    EDGE_BOTH = 2,
};

enum {
    TRIGGER_FRAME_MID = 0,
    TRIGGER_FRAME_START = 1,
    TRIGGER_FRAME_END = 2,
};

class AsyncCommand {
    // variable
  private:
    PropertyUtility& m_propertyUtility;
    dataInfo_t m_dataInfo;
    bool m_isShowDebugMsg;
    vector<pair<uint16_t, uint32_t> > getThresholdPoint(int channelIndex, const vector<vector<double> >& data, int edgeType, double threshold, double deviation);
    vector<pair<uint16_t, uint32_t> > getTriggerMaskPoint(uint32_t dataCount, vector<pair<uint16_t, uint32_t> > thresholdPoint);
    double m_preValue;
    uint16_t m_trigger;
    int m_triggerStatus;
    uint64_t m_sendedPoints;
    uint64_t m_tiggerMaskPoints;
    uint64_t m_recordTimeoutPoints;
    uint64_t m_pointsAfterLastTrigger;
    uint32_t getThresholdFirstLargeThan(vector<pair<uint16_t, uint32_t> > v, uint32_t d);
    uint32_t getPositionInRange(uint32_t pos, uint32_t count);
    // accessor
  public:
    AsyncCommand();
    ~AsyncCommand();

    inline void setDataInfo(dataInfo_t* pDataInfo) {
        m_dataInfo.chanStart = pDataInfo->chanStart;
        m_dataInfo.chanCount = pDataInfo->chanCount;
        m_dataInfo.dataSize = pDataInfo->dataSize;
        m_dataInfo.pDataBuf = pDataInfo->pDataBuf;
    }
    inline dataInfo_t* getDataInfo() {
        return &m_dataInfo;
    }
    inline void setPreValue(double preValue) {
        m_preValue = preValue;
    }
    inline double getPreValue() {
        return m_preValue;
    }
    inline void setTrigger(uint16_t trigger) {
        m_trigger = trigger;
    }
    inline int getTiggerStatus() {
        return m_triggerStatus;
    }
    inline void setTriggerStatus(int triggerStatus) {
        m_triggerStatus = triggerStatus;
    }
    inline void setSendedPoints(uint64_t p) {
        m_sendedPoints = p;
    }
    inline uint64_t getSendedPoints() {
        return m_sendedPoints;
    }
    inline void setPointsAfterLastTrigger(uint64_t p) {
        m_pointsAfterLastTrigger = p;
    }
    inline uint64_t getPointsAfterLastTrigger() {
        return m_pointsAfterLastTrigger;
    }
    void pipeOut(void* buf, size_t size, uint32_t count);
    int execute();
};

#endif // _ASYNCCOMMAND_HPP_
