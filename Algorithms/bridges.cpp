#include <algorithm>
#include <iostream>
#include <map>
#include <vector>

/*Найти мосты.*/

class Graph {
 public:
  explicit Graph(size_t number) : edges_(number) {}

  void AddEdge(size_t from, size_t too, size_t number) {
    edges_[from].push_back({too, number});
  }

  std::vector<std::pair<size_t, size_t>> Neighbours(size_t node) const {
    return edges_[node];
  }

  size_t Size() const { return edges_.size(); }

 private:
  std::vector<std::vector<std::pair<size_t, size_t>>> edges_;
};

struct BridgesData {
  std::vector<size_t> bridges;
  std::map<std::pair<size_t, size_t>, size_t> repeat_edges;
};

struct NodesData {
  std::vector<int> colors;
  std::vector<size_t> tin;
  std::vector<size_t> return_time;

  explicit NodesData(size_t node_number)
      : colors(node_number + 1),
        tin(node_number + 1),
        return_time(node_number + 1) {}
};

void DFS(const Graph& graph, std::pair<size_t, size_t> edge, size_t& in_time,
         BridgesData& bridges_data, NodesData& nodes_data) {
  nodes_data.colors[edge.second] = 1;
  nodes_data.tin[edge.second] = in_time;
  nodes_data.return_time[edge.second] = in_time;
  size_t renew_return_time = in_time;
  ++in_time;
  for (auto next : graph.Neighbours(edge.second)) {
    if (nodes_data.colors[next.first] == 1 and next.first != edge.first) {
      renew_return_time =
          std::min(renew_return_time, nodes_data.tin[next.first]);
    }
  }
  for (auto next : graph.Neighbours(edge.second)) {
    if (nodes_data.colors[next.first] == 0) {
      DFS(graph, {edge.second, next.first}, in_time, bridges_data, nodes_data);
      nodes_data.return_time[edge.second] =
          std::min(nodes_data.return_time[edge.second],
                   nodes_data.return_time[next.first]);
      if (nodes_data.return_time[next.first] == nodes_data.tin[next.first] and
          bridges_data.repeat_edges[{edge.second, next.first}] == 1) {
        bridges_data.bridges.push_back(next.second);
      }
    }
  }
  nodes_data.return_time[edge.second] =
      std::min(nodes_data.return_time[edge.second], renew_return_time);
}

void FindBridges(size_t node_number, const Graph& graph,
                 BridgesData& bridges_data) {
  size_t in_time = 0;
  NodesData nodes_data(node_number);
  for (size_t i = 1; i <= node_number; ++i) {
    if (nodes_data.colors[i] == 0) {
      DFS(graph, {0, i}, in_time, bridges_data, nodes_data);
    }
  }

  sort(bridges_data.bridges.begin(), bridges_data.bridges.end());
}

int main() {
  size_t node_number;
  size_t edge_number;
  std::cin >> node_number >> edge_number;
  Graph graph(node_number + 1);
  BridgesData bridges_data;
  size_t number = 0;
  for (size_t i = 0; i < edge_number; ++i) {
    size_t first;
    size_t second;
    ++number;
    std::cin >> first >> second;
    graph.AddEdge(first, second, number);
    graph.AddEdge(second, first, number);
    ++bridges_data.repeat_edges[{first, second}];
    ++bridges_data.repeat_edges[{second, first}];
  }

  FindBridges(node_number, graph, bridges_data);
  std::cout << bridges_data.bridges.size() << '\n';
  for (size_t edge : bridges_data.bridges) {
    std::cout << edge << ' ';
  }
}