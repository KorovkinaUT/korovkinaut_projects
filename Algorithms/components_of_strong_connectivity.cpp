#include <algorithm>
#include <iostream>
#include <vector>

/*Найти компоненты сильной связности. Для каждой вершины вывести какой КСС
  она пренадлежит.*/

class Graph {
 public:
  explicit Graph(size_t number) : edges_(number) {}

  void AddEdge(size_t from, size_t too) { edges_[from].push_back(too); }

  std::vector<size_t> Neighbours(size_t node) const { return edges_[node]; }

  size_t Size() const { return edges_.size(); }

 private:
  std::vector<std::vector<size_t>> edges_;
};

Graph Reverse(const Graph& graph) {
  Graph reversed(graph.Size());
  for (size_t i = 1; i < graph.Size(); ++i) {
    for (size_t next : graph.Neighbours(i)) {
      reversed.AddEdge(next, i);
    }
  }
  return reversed;
}

struct NodesData {
  std::vector<int> colors;
  std::vector<size_t> tout;

  explicit NodesData(size_t node_number)
      : colors(node_number + 1), tout(node_number + 1) {}
};

void DFS(const Graph& graph, NodesData& nodes_data, size_t node,
         size_t& out_time, std::vector<size_t>& component) {
  nodes_data.colors[node] = 1;
  for (size_t next : graph.Neighbours(node)) {
    if (nodes_data.colors[next] == 0) {
      component.push_back(next);
      DFS(graph, nodes_data, next, out_time, component);
    }
  }
  nodes_data.colors[node] = 2;
  nodes_data.tout[node] = out_time;
  ++out_time;
}

void TopologicalSort(const NodesData& nodes_data, std::vector<size_t>& sorted) {
  std::vector<std::pair<size_t, size_t>> to_sort(nodes_data.tout.size() - 1);
  for (size_t i = 1; i < nodes_data.tout.size(); ++i) {
    to_sort[i - 1] = {nodes_data.tout[i], i};
  }
  sort(to_sort.rbegin(), to_sort.rend());
  for (size_t i = 0; i < sorted.size(); ++i) {
    sorted[i] = to_sort[i].second;
  }
}

void StronglyConnectedComponents(size_t node_number, const Graph& graph,
                                 std::vector<size_t>& components) {
  NodesData nodes_data(node_number);
  std::vector<size_t> component;
  size_t out_time = 0;
  for (size_t i = 1; i <= node_number; ++i) {
    if (nodes_data.colors[i] == 0) {
      component.clear();
      DFS(graph, nodes_data, i, out_time, component);
    }
  }
  std::vector<size_t> sorted(node_number);
  TopologicalSort(nodes_data, sorted);

  out_time = 0;
  Graph reversed = Reverse(graph);
  NodesData reversed_nodes_data(node_number);
  size_t current_component = 0;
  for (size_t node : sorted) {
    if (reversed_nodes_data.colors[node] == 0) {
      component.clear();
      ++current_component;
      component.push_back(node);
      DFS(reversed, reversed_nodes_data, node, out_time, component);
    }
    for (size_t node : component) {
      components[node] = current_component;
    }
  }
}

int main() {
  size_t node_number;
  size_t edge_number;
  std::cin >> node_number >> edge_number;
  Graph graph(node_number + 1);
  for (size_t i = 0; i < edge_number; ++i) {
    size_t from;
    size_t too;
    std::cin >> from >> too;
    graph.AddEdge(from, too);
  }

  std::vector<size_t> components(node_number + 1);
  StronglyConnectedComponents(node_number, graph, components);
  std::cout << *std::max_element(components.begin(), components.end()) << '\n';
  for (size_t i = 1; i <= node_number; ++i) {
    std::cout << components[i] << ' ';
  }
}