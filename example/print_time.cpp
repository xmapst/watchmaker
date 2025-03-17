#include <iostream>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <chrono>
#include <ctime>
#include <sys/time.h>
#include <unistd.h>
#include <iomanip>
#include <sstream>
#include <array>

// 同步相关的全局变量
std::mutex cv_mutex;
std::condition_variable cv;
std::array<std::string, 4> outputs;  // 存放4个线程的输出
int ready_count = 0;                // 已经采集完成的线程数
bool start_cycle = false;           // 控制各线程开始采集的标志

// 用于主线程打印输出时的互斥保护
std::mutex cout_mutex;

// 工作线程：利用 clock_gettime 获取时间，并将格式化字符串保存到 outputs[index]
void workerClockGetTime(int index) {
    while (true) {
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return start_cycle; });
        }
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        struct tm tm_ts;
        localtime_r(&ts.tv_sec, &tm_ts);
        std::ostringstream oss;
        oss << std::put_time(&tm_ts, "%Y-%m-%d %H:%M:%S")
            << '.' << std::setw(9) << std::setfill('0') << ts.tv_nsec;
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            outputs[index] = std::string("clock_gettime()") + "\t\t" +
                             std::to_string(ts.tv_sec) + " s, " +
                             std::to_string(ts.tv_nsec) + " ns" + "\t" + oss.str();
            ready_count++;
        }
        cv.notify_all();
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return !start_cycle; });
        }
    }
}

// 工作线程：利用 time() 获取时间
void workerTime(int index) {
    while (true) {
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return start_cycle; });
        }
        time_t t = time(nullptr);
        struct tm tm_time;
        localtime_r(&t, &tm_time);
        std::ostringstream oss;
        oss << std::put_time(&tm_time, "%Y-%m-%d %H:%M:%S");
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            outputs[index] = std::string("time()") + "\t\t\t" +
                             std::to_string(t) + " s" + "\t\t\t" + oss.str();
            ready_count++;
        }
        cv.notify_all();
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return !start_cycle; });
        }
    }
}

// 工作线程：利用 gettimeofday 获取时间
void workerGettimeofday(int index) {
    while (true) {
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return start_cycle; });
        }
        struct timeval tv;
        gettimeofday(&tv, nullptr);
        struct tm tm_tv;
        localtime_r(&tv.tv_sec, &tm_tv);
        std::ostringstream oss;
        oss << std::put_time(&tm_tv, "%Y-%m-%d %H:%M:%S")
            << '.' << std::setw(6) << std::setfill('0') << tv.tv_usec;
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            outputs[index] = std::string("gettimeofday()") + "\t\t" +
                             std::to_string(tv.tv_sec) + " s, " +
                             std::to_string(tv.tv_usec) + " us" + "\t\t" + oss.str();
            ready_count++;
        }
        cv.notify_all();
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return !start_cycle; });
        }
    }
}

// 工作线程：利用 std::chrono 获取时间
void workerChrono(int index) {
    while (true) {
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return start_cycle; });
        }
        auto now = std::chrono::system_clock::now();
        auto now_time_t = std::chrono::system_clock::to_time_t(now);
        auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
            now.time_since_epoch()) % 1000;
        struct tm tm_chrono;
        localtime_r(&now_time_t, &tm_chrono);
        std::ostringstream oss;
        oss << std::put_time(&tm_chrono, "%Y-%m-%d %H:%M:%S")
            << '.' << std::setw(3) << std::setfill('0') << ms.count();
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            outputs[index] = std::string("std::chrono") + "\t\t(system_clock)" + "\t\t\t" + oss.str();
            ready_count++;
        }
        cv.notify_all();
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return !start_cycle; });
        }
    }
}

int main() {
    // 启动4个工作线程，并分别指定输出在 outputs 数组中的位置
    std::thread t1(workerClockGetTime, 0);
    std::thread t2(workerTime, 1);
    std::thread t3(workerGettimeofday, 2);
    std::thread t4(workerChrono, 3);

    // 表头只打印一次，使用 Tab 分隔各个字段
    {
        std::lock_guard<std::mutex> lock(cout_mutex);
        std::cout << "方法\t\t\t原始数据\t\t\t格式化时间" << std::endl;
        std::cout << "--------------------------------------------------------------------------------" << std::endl;
    }

    // 主线程循环：每个周期统一打印所有线程的数据和分割符
    while (true) {
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            start_cycle = true;
        }
        cv.notify_all();

        // 等待所有线程采集完成（4个线程都更新后）
        {
            std::unique_lock<std::mutex> lock(cv_mutex);
            cv.wait(lock, []{ return ready_count == 4; });
        }

        // 统一打印输出（数据各字段之间使用 Tab 分隔）
        {
            std::lock_guard<std::mutex> lock(cout_mutex);
            for (const auto &line : outputs) {
                std::cout << line << std::endl;
            }
            std::cout << "--------------------------------------------------------------------------------" << std::endl;
        }

        // 重置标志，结束本周期，通知各线程准备下一个周期
        {
            std::lock_guard<std::mutex> lock(cv_mutex);
            start_cycle = false;
            ready_count = 0;
        }
        cv.notify_all();

        sleep(1);
    }

    // 由于各线程都是无限循环，join 永远不会返回
    t1.join();
    t2.join();
    t3.join();
    t4.join();
    return 0;
}
