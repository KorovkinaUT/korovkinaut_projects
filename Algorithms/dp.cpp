#include <iostream>
#include <vector>

/* Условие: Карлу необходимо выполнить для мистера Правды N поручений,
            каждое из них характеризуется двумя числами:
            необходимое число ресурсов m и награда c.
            Сиджею негде набирать ресурсы, так что он ограничен
            M единицами ресурсов. Какие задания он может выполнить,
            чтобы максимизировать награду?
   Динамика: dp[i][j] - максимальная награда если выполнить задания с номерами
   <= i и потратить ресурса <= j.*/

void FindTasks(std::vector<size_t>& tasks, size_t number,
               std::vector<std::vector<size_t> >& dp, size_t resource,
               std::vector<size_t>& resources) {
  size_t first_index = number;
  int second_index = resource;
  while (first_index > 0 and second_index > 0) {
    if (dp[first_index][second_index] == dp[first_index - 1][second_index]) {
      --first_index;
    } else {
      tasks.push_back(first_index);
      second_index -= resources[first_index];
      --first_index;
    }
  }
}

void MaxRewardTasks(size_t number, std::vector<size_t>& resources,
                    size_t resource, std::vector<size_t>& rewards,
                    std::vector<size_t>& tasks) {
  std::vector<std::vector<size_t> > dp(number + 1,
                                       std::vector<size_t>(resource + 1));
  for (size_t i = 1; i <= number; ++i) {
    for (size_t j = 1; j <= resource; ++j) {
      dp[i][j] = dp[i - 1][j];
      if (j >= resources[i]) {
        dp[i][j] = std::max(dp[i][j], dp[i - 1][j - resources[i]] + rewards[i]);
      }
    }
  }
  FindTasks(tasks, number, dp, resource, resources);
}

int main() {
  size_t number;
  size_t resource;
  std::cin >> number >> resource;
  std::vector<size_t> resources(number + 1);
  std::vector<size_t> rewards(number + 1);
  for (size_t i = 0; i < number; ++i) {
    std::cin >> resources[i + 1];
  }
  for (size_t i = 0; i < number; ++i) {
    std::cin >> rewards[i + 1];
  }

  std::vector<size_t> tasks;
  MaxRewardTasks(number, resources, resource, rewards, tasks);
  for (int i = tasks.size() - 1; i >= 0; --i) {
    std::cout << tasks[i] << '\n';
  }
}