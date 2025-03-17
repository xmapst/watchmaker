#include <iostream>
#include <ctime>
#include <sys/time.h>
#include <unistd.h>
#include <iomanip>
#include <sstream>
#include <chrono>

int main() {
    // 打印表头（仅打印一次）
    std::cout << std::left << std::setw(20) << "方法"
              << std::setw(40) << "原始数据"
              << "格式化时间" << std::endl;
    std::cout << "--------------------------------------------------------------------------------" << std::endl;

    while (true) {
        // 1. 使用 clock_gettime 获取时间（秒和纳秒）
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        struct tm tm_ts;
        localtime_r(&ts.tv_sec, &tm_ts);
        std::ostringstream oss_ts;
        oss_ts << std::put_time(&tm_ts, "%Y-%m-%d %H:%M:%S");
        std::string formatted_ts = oss_ts.str();

        // 2. 使用 time() 获取时间（秒）
        time_t t = time(nullptr);
        struct tm tm_time;
        localtime_r(&t, &tm_time);
        std::ostringstream oss_time;
        oss_time << std::put_time(&tm_time, "%Y-%m-%d %H:%M:%S");
        std::string formatted_time = oss_time.str();

        // 3. 使用 gettimeofday 获取时间（秒和微秒）
        struct timeval tv;
        gettimeofday(&tv, nullptr);
        struct tm tm_tv;
        localtime_r(&tv.tv_sec, &tm_tv);
        std::ostringstream oss_tv;
        oss_tv << std::put_time(&tm_tv, "%Y-%m-%d %H:%M:%S");
        std::string formatted_tv = oss_tv.str();

        // 4. 使用 std::chrono 获取时间（系统时钟，秒和毫秒）
        auto now = std::chrono::system_clock::now();
        auto now_time_t = std::chrono::system_clock::to_time_t(now);
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(now.time_since_epoch()) % 1000;
        struct tm tm_chrono;
        localtime_r(&now_time_t, &tm_chrono);
        std::ostringstream oss_chrono;
        oss_chrono << std::put_time(&tm_chrono, "%Y-%m-%d %H:%M:%S")
                   << '.' << std::setfill('0') << std::setw(3) << ms.count();
        std::string formatted_chrono = oss_chrono.str();

        // 输出 clock_gettime() 的结果
        std::cout << std::left << std::setw(20) << "clock_gettime()"
                  << std::setw(40) << (std::to_string(ts.tv_sec) + " s, " + std::to_string(ts.tv_nsec) + " ns")
                  << formatted_ts << std::endl;

        // 输出 time() 的结果
        std::cout << std::left << std::setw(20) << "time()"
                  << std::setw(40) << (std::to_string(t) + " s")
                  << formatted_time << std::endl;

        // 输出 gettimeofday() 的结果
        std::cout << std::left << std::setw(20) << "gettimeofday()"
                  << std::setw(40) << (std::to_string(tv.tv_sec) + " s, " + std::to_string(tv.tv_usec) + " us")
                  << formatted_tv << std::endl;

        // 输出 std::chrono 的结果
        std::cout << std::left << std::setw(20) << "std::chrono"
                  << std::setw(40) << "(system_clock)"
                  << formatted_chrono << std::endl;

        std::cout << "--------------------------------------------------------------------------------" << std::endl;

        // 休眠 1 秒
        sleep(1);
    }

    return 0;
}