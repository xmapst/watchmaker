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
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <fcntl.h>
#include <sys/epoll.h>
#include <cstring>

// 获取格式化时间的函数
template<typename TimeFunc>
std::string getFormattedTime(TimeFunc timeFunc) {
    return timeFunc();
}

auto clockGetTimeFunc = []() -> std::string {
    timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    tm tm_ts;
    localtime_r(&ts.tv_sec, &tm_ts);
    std::ostringstream oss;
    oss << std::put_time(&tm_ts, "%Y-%m-%d %H:%M:%S")
        << '.' << std::setw(9) << std::setfill('0') << ts.tv_nsec;
    return oss.str();
};

auto timeFunc = []() -> std::string {
    time_t t = time(nullptr);
    tm tm_time;
    localtime_r(&t, &tm_time);
    std::ostringstream oss;
    oss << std::put_time(&tm_time, "%Y-%m-%d %H:%M:%S");
    return oss.str();
};

auto gettimeofdayFunc = []() -> std::string {
    timeval tv;
    gettimeofday(&tv, nullptr);
    tm tm_tv;
    localtime_r(&tv.tv_sec, &tm_tv);
    std::ostringstream oss;
    oss << std::put_time(&tm_tv, "%Y-%m-%d %H:%M:%S")
        << '.' << std::setw(6) << std::setfill('0') << tv.tv_usec;
    return oss.str();
};

// epoll 服务器线程函数（返回格式化时间 JSON，每秒发送一次）
void epollServer() {
    const int PORT = 8080;
    const int MAX_EVENTS = 10;

    int epoll_fd = epoll_create1(0);
    if (epoll_fd == -1) {
        perror("epoll_create1");
        return;
    }

    int listen_sock = socket(AF_INET, SOCK_STREAM | SOCK_NONBLOCK, 0);
    if (listen_sock == -1) {
        perror("socket");
        return;
    }

    sockaddr_in server_addr{};
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(PORT);

    if (bind(listen_sock, (sockaddr*)&server_addr, sizeof(server_addr)) == -1) {
        perror("bind");
        close(listen_sock);
        return;
    }

    if (listen(listen_sock, SOMAXCONN) == -1) {
        perror("listen");
        close(listen_sock);
        return;
    }

    epoll_event event{};
    event.events = EPOLLIN;
    event.data.fd = listen_sock;
    if (epoll_ctl(epoll_fd, EPOLL_CTL_ADD, listen_sock, &event) == -1) {
        perror("epoll_ctl");
        close(listen_sock);
        return;
    }

    epoll_event events[MAX_EVENTS];

    while (true) {
        int num_events = epoll_wait(epoll_fd, events, MAX_EVENTS, -1);
        if (num_events == -1) {
            if (errno == EINTR)
                continue;
            perror("epoll_wait");
            break;
        }
        for (int i = 0; i < num_events; ++i) {
            if (events[i].data.fd == listen_sock) {
                sockaddr_in client_addr{};
                socklen_t client_len = sizeof(client_addr);
                int client_fd = accept(listen_sock, (sockaddr*)&client_addr, &client_len);
                if (client_fd == -1) {
                    if (errno != EAGAIN && errno != EWOULDBLOCK)
                        perror("accept");
                    continue;
                }
                // 设置非阻塞模式
                fcntl(client_fd, F_SETFL, O_NONBLOCK);

                // 为每个连接启动一个线程，每秒发送一次时间 JSON
                std::thread([client_fd]() {
                    while (true) {
                        std::ostringstream json_response;
                        json_response << "{\"clock_gettime\": \"" << clockGetTimeFunc() << "\",";
                        json_response << "\"time\": \"" << timeFunc() << "\",";
                        json_response << "\"gettimeofday\": \"" << gettimeofdayFunc() << "\"}\n";
                        std::string response = json_response.str();
                        ssize_t sent = send(client_fd, response.c_str(), response.size(), 0);
                        if (sent == -1) {
                            perror("send");
                            break;
                        }
                        std::this_thread::sleep_for(std::chrono::seconds(1));
                    }
                    close(client_fd);
                }).detach();
            }
        }
    }
    close(listen_sock);
    close(epoll_fd);
}

int main() {
    std::thread server_thread(epollServer);
    server_thread.detach();
    // 主线程每秒打印三种时间格式
    while (true) {
        std::cout << "clock_gettime: " << clockGetTimeFunc()
                  << ", time: " << timeFunc()
                  << ", gettimeofday: " << gettimeofdayFunc() << std::endl;
        sleep(1);
    }
    return 0;
}
