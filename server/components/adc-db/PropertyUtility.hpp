#ifndef _PROPERTY_UTILITY_HPP_
#define _PROPERTY_UTILITY_HPP_
#include <string>
#include <stdint.h>
#include <sys/stat.h>
#include <map>
#include "Singleton.hpp"
#include "ChangeSettingUtility.hpp"
using namespace std;

typedef struct device {
    wstring Description;
    uint32_t StartChannel;
    uint32_t SampleChannelCount;
    uint32_t ChannelCountMax;
    uint32_t IntervalCount;
    //uint32_t SampleBuffSize;
    int32_t ValueRange; 
    uint32_t SignalType; 
    double SamplingRate; 
} device_t;

typedef struct softTrigger {
    uint32_t IntervalMs;
    double Threshold;
    double ThresholdDeviation;
    float ThresholdMaskTime;
    uint32_t ChannelIndex; // 0 origin
    int32_t EdgeType;
} softTrigger_t;

typedef struct softCtrl {
    bool IsOutputSummaryData;
    bool IsOutputRawData;
    int SoftTriggerType;
    softTrigger_t SoftTrigger;
    float RecordTimeout;
    bool IsDumpData;
    string DumpDataPath;
    bool IsShowDebugLog;
} softCtrl_t;

typedef struct SummaryData {
    uint32_t DataCountPerChannel;
    uint32_t Type;
    uint32_t DataSize;
    double ExpectMax;
    double ExpectMin;
} summaryData_t;

typedef struct FFTData {
    uint32_t Type;
    uint16_t ExpectMax;
    uint16_t ExpectMin;
} FFTData_t;

typedef struct RawData {
    uint32_t Type;
} rawData_t;

typedef struct adcDataFmt {
    summaryData_t SummaryData;
    FFTData_t FFTData;
    rawData_t RawData;
} adcDataFmt_t;

enum {
    SOFT_TRIGGER_TYPE_NONE = 0,
    SOFT_TRIGGER_TYPE_TIME_INTERVAL = 1,
    SOFT_TRIGGER_TYPE_THRESHOLD = 2,
};

class PropertyUtility : public Singleton<PropertyUtility>
{
public:
    PropertyUtility();
    virtual ~PropertyUtility();
    PropertyUtility(const PropertyUtility&);

private:
    friend Singleton<PropertyUtility>;
    friend class std::unique_ptr<PropertyUtility>;
    PropertyUtility& operator =(const PropertyUtility&);
    inline PropertyUtility& getInstance(void)
{
        return Singleton<PropertyUtility>::GetInstance();
    }
    // variable
    device_t m_device;
    map<string, int> m_valueRangeMap;
    softCtrl_t m_softCtrl;
    adcDataFmt_t m_adcDataFmt;
    string m_propertyFileName;
    uint64_t m_tiggerMaskPoints;
    uint64_t m_recordTimeoutPoints;
    // accessor
public:
    void setPropertyFileName(char* propertyFileName);
    void getProperties();
    bool isExistPropertyFile();

    inline device_t & getDeviceInfomation()
    {
        return (m_device);
    }

    inline softCtrl_t & getSoftCtrl()
    {
        return (m_softCtrl);
    }
    inline adcDataFmt_t & getAdcDataFmt()
    {
        return (m_adcDataFmt);
    }
    inline uint64_t getTiggerMaskPoints()
    {
        return (m_tiggerMaskPoints);
    }
    inline uint64_t getRecordTimeoutPoints()
    {
        return (m_recordTimeoutPoints);
    }

    void notifyChangedSetting(ChangeSettingUtility *p);
    void saveProperties();
};

#endif // _PROPERTY_UTILITY_HPP_
