#include <algorithm>
#include <iostream>
#include <queue>
#include <string>
#include <vector>

const size_t cAlphabetSize = 26;
const size_t cIndexShift = 97;

size_t CharNumber(char letter) { return static_cast<size_t>(letter); }

struct Node {
  std::vector<int> edges;
  size_t word;
  bool is_word;
  bool reversed;

  Node() : edges(cAlphabetSize, -1), is_word(false), reversed(true){};
};

class Trie {
 private:
  std::vector<Node*> nodes_;
  void DeleteTrie(size_t current);

 public:
  Trie() : nodes_(1, new Node()){};
  const Node* operator[](size_t index) const { return nodes_[index]; }
  Node* operator[](size_t index) { return nodes_[index]; }
  Node* AddWord(const std::string& word, std::vector<size_t>& subwords);
  size_t Size() const { return nodes_.size(); }
  ~Trie() { DeleteTrie(0); }
};

void Trie::DeleteTrie(size_t current) {
  for (size_t i = 0; i < cAlphabetSize; ++i) {
    if (nodes_[current]->edges[i] != -1) {
      DeleteTrie(nodes_[current]->edges[i]);
    }
  }
  delete nodes_[current];
}

Node* Trie::AddWord(const std::string& word, std::vector<size_t>& subwords) {
  Node* current_node = nodes_[0];
  for (size_t i = 0; i < word.length(); ++i) {
    if (current_node->edges[CharNumber(word[i]) - cIndexShift] == -1) {
      Node* new_node = new Node();
      nodes_.push_back(new_node);
      current_node->edges[CharNumber(word[i]) - cIndexShift] =
          nodes_.size() - 1;
    }
    current_node =
        nodes_[current_node->edges[CharNumber(word[i]) - cIndexShift]];
    if (current_node->is_word and !current_node->reversed) {
      subwords.push_back(current_node->word);
    }
  }
  current_node->is_word = true;
  return current_node;
}

bool IsPalindrom(const std::string& string) {
  if (string.empty()) {
    return true;
  }
  for (size_t i = 0; i < string.size() / 2; ++i) {
    if (string[i] != string[string.size() - i - 1]) {
      return false;
    }
  }
  return true;
}

void DirectOrder(std::vector<std::string>& strings,
                 std::vector<std::pair<size_t, size_t>>& pairs) {
  Trie trie;
  std::vector<size_t> subwords;
  for (size_t i = 0; i < strings.size(); ++i) {
    Node* node = trie.AddWord(strings[i], subwords);
    node->is_word = true;
    node->reversed = false;
    node->word = i;
    subwords.clear();
    std::reverse(strings[i].begin(), strings[i].end());
  }

  subwords.clear();
  for (size_t i = 0; i < strings.size(); ++i) {
    trie.AddWord(strings[i], subwords);

    for (auto subword : subwords) {
      if (i != subword &&
          IsPalindrom(strings[i].substr(strings[subword].size()))) {
        pairs.push_back({subword, i});
      }
    }

    subwords.clear();
  }
}

void ReversedCount(std::vector<std::string>& strings,
                   std::vector<std::pair<size_t, size_t>>& pairs) {
  Trie trie;
  std::vector<size_t> subwords;
  for (size_t i = 0; i < strings.size(); ++i) {
    Node* node = trie.AddWord(strings[i], subwords);
    node->is_word = true;
    node->reversed = false;
    node->word = i;
    subwords.clear();
    std::reverse(strings[i].begin(), strings[i].end());
  }

  subwords.clear();
  for (size_t i = 0; i < strings.size(); ++i) {
    trie.AddWord(strings[i], subwords);

    for (auto subword : subwords) {
      if (i != subword &&
          IsPalindrom(strings[i].substr(strings[subword].size())) &&
          strings[subword].size() < strings[i].size()) {
        pairs.push_back({i, subword});
      }
    }

    subwords.clear();
  }
}

int main() {
  size_t number;
  std::cin >> number;
  std::vector<std::string> strings(number);
  for (size_t i = 0; i < number; ++i) {
    std::cin >> strings[i];
  }
  std::vector<std::pair<size_t, size_t>> pairs;
  DirectOrder(strings, pairs);
  ReversedCount(strings, pairs);
  std::cout << pairs.size() << '\n';
  for (auto pair : pairs) {
    std::cout << pair.first + 1 << ' ' << pair.second + 1 << '\n';
  }
}