#include <iostream>
#include <vector>

/*Условие: реализовать AVL дерево с запросами добавление элемента
           и найти минимальный ключ не меньший i.*/

class AVLTree {
 private:
  struct Node {
    size_t data;
    size_t height;
    Node* left;
    Node* right;
    Node(size_t data) : data(data), height(1), left(nullptr), right(nullptr) {}
  };

  static std::pair<size_t, size_t> ChildrenHeights(Node* node);
  static void ChangeHeightNode(Node* node);
  static void ChangeHeight(Node* node);
  static int DeltaHeight(Node* node);
  static void SmallLeftRotation(Node* node);
  static void SmallRightRotation(Node* node);
  static void BigLeftRotation(Node* node);
  static void BigRightRotation(Node* node);
  static void Repair(Node* node);

 public:
  Node* tree = nullptr;
  AVLTree() = default;
  ~AVLTree() = default;

  bool Find(Node* node, size_t data);
  void Insert(Node* node, size_t data);
  int Next(Node* node, size_t data, size_t current_answer);
  void DeleteTree(Node* node);
};

std::pair<size_t, size_t> AVLTree::ChildrenHeights(Node* node) {
  size_t left_height = node->left == nullptr ? 0 : node->left->height;
  size_t right_height = node->right == nullptr ? 0 : node->right->height;
  return {left_height, right_height};
}

void AVLTree::ChangeHeightNode(Node* node) {
  if (node == nullptr) {
    return;
  }
  auto [left_height, right_height] = ChildrenHeights(node);
  node->height = std::max(left_height, right_height) + 1;
}

void AVLTree::ChangeHeight(Node* node) {
  ChangeHeightNode(node->left);
  ChangeHeightNode(node->right);
  ChangeHeightNode(node);
}

int AVLTree::DeltaHeight(Node* node) {
  if (node == nullptr) {
    return 0;
  }
  auto [left_height, right_height] = ChildrenHeights(node);
  return left_height - right_height;
}

void AVLTree::SmallLeftRotation(Node* node) {
  std::swap(node->data, node->right->data);
  Node* right = node->right;
  node->right = right->right;
  right->right = right->left;
  right->left = node->left;
  node->left = right;
  ChangeHeight(node);
}

void AVLTree::SmallRightRotation(Node* node) {
  std::swap(node->data, node->left->data);
  Node* left = node->left;
  node->left = left->left;
  left->left = left->right;
  left->right = node->right;
  node->right = left;
  ChangeHeight(node);
}

void AVLTree::BigLeftRotation(Node* node) {
  SmallRightRotation(node->right);
  SmallLeftRotation(node);
  ChangeHeight(node);
}

void AVLTree::BigRightRotation(Node* node) {
  SmallLeftRotation(node->left);
  SmallRightRotation(node);
  ChangeHeight(node);
}

void AVLTree::Repair(Node* node) {
  ChangeHeightNode(node);
  if (DeltaHeight(node) == -2) {
    if (DeltaHeight(node->right) <= 0) {
      SmallLeftRotation(node);
    } else {
      BigLeftRotation(node);
    }
  } else if (DeltaHeight(node) == 2) {
    if (DeltaHeight(node->left) >= 0) {
      SmallRightRotation(node);
    } else {
      BigRightRotation(node);
    }
  }
}

bool AVLTree::Find(Node* node, size_t data) {
  if (node == nullptr) {
    return false;
  }
  if (node->data == data) {
    return true;
  }
  if (node->data > data) {
    return Find(node->left, data);
  }
  return Find(node->right, data);
}

void AVLTree::Insert(Node* node, size_t data) {
  if (tree == nullptr) {
    tree = new Node(data);
    return;
  }
  if (node->data < data and node->right == nullptr) {
    node->right = new Node(data);
    Repair(node);
    return;
  }
  if (node->data > data and node->left == nullptr) {
    node->left = new Node(data);
    Repair(node);
    return;
  }
  if (node->data < data) {
    Insert(node->right, data);
  } else {
    Insert(node->left, data);
  }
  Repair(node);
}

int AVLTree::Next(Node* node, size_t data, size_t current_answer) {
  if (node == nullptr) {
    return current_answer;
  }
  if (node->data < data) {
    return Next(node->right, data, current_answer);
  }
  current_answer = std::min(current_answer, node->data);
  return Next(node->left, data, current_answer);
}

void AVLTree::DeleteTree(Node* node) {
  if (node == nullptr) {
    return;
  }
  DeleteTree(node->left);
  DeleteTree(node->right);
  delete node;
}

int main() {
  size_t number;
  std::cin >> number;
  char previous_request = '+';
  int previous_result;
  const int kMax = 1000000001;
  AVLTree tree;
  for (size_t i = 0; i < number; ++i) {
    char request;
    size_t data;
    std::cin >> request >> data;
    if (request == '+') {
      if (previous_request == '?') {
        data = (data + previous_result) % (kMax - 1);
      }
      if (not tree.Find(tree.tree, data)) {
        tree.Insert(tree.tree, data);
      }
    } else if (request == '?') {
      previous_result = tree.Next(tree.tree, data, kMax);
      if (previous_result == kMax) {
        previous_result = -1;
      }
      std::cout << previous_result << std::endl;
    }
    previous_request = request;
  }

  tree.DeleteTree(tree.tree);
}