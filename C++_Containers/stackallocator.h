#include <memory>
#include <iostream>

template <std::size_t N>
class StackStorage {
private:
  void* top_;
  char buffer_[N];

public:
  StackStorage(): top_(buffer_), buffer_() {};
  StackStorage(const StackStorage&) = delete;
  StackStorage& operator=(const StackStorage&) = delete;
  ~StackStorage() = default;

  template<typename T>
  T* allocate(std::size_t number) {
    std::size_t buffer_size = (buffer_ + N) - static_cast<char*>(top_);
    if (std::align(alignof(T), number * sizeof(T), top_, buffer_size)) {
      T* ptr = reinterpret_cast<T*>(top_);
      top_ = static_cast<char*>(top_) + number * sizeof(T);
      return ptr;
    }
    return nullptr;
  }
};

template <typename T, std::size_t N>
class StackAllocator {
private:
  StackStorage<N>* storage_;

public:
  using value_type = T;

  template <typename U>
  struct rebind {
    using other = StackAllocator<U, N>;
  };

  StackAllocator(StackStorage<N>& storage): storage_(&storage) {};
  ~StackAllocator() = default;

  StackStorage<N>* storage_ptr() const { return storage_; }

  template <typename U>
  StackAllocator(const StackAllocator<U, N>& other): storage_(other.storage_ptr()) {};
  template <typename U>
  StackAllocator& operator=(const StackAllocator<U, N>& other) {
    storage_ = other.storage_ptr();
    return *this;
  }
  template <typename U>
  bool operator==(const StackAllocator<U, N>& other) const {
    return storage_ == other.storage_ptr();
  }
  template <typename U>
  bool operator!=(const StackAllocator<U, N>& other) const { return !(*this == other); }

  T* allocate(std::size_t number) const { return storage_->template allocate<T>(number); }
  void deallocate(T*, std::size_t) const {};
};

template <typename T, typename Allocator = std::allocator<T>>
class List {
private:
  struct BasicalNode;
  struct Node;
  template <bool is_const>
  struct general_iterator;
  using NodeAllocator = typename std::allocator_traits<Allocator>::template rebind_alloc<Node>;
  using NodeTraits = std::allocator_traits<NodeAllocator>;

  BasicalNode fake_;
  std::size_t list_size_;
  [[no_unique_address]] NodeAllocator allocator_;

  void swap_all(List& other);

public:
  using allocator_type = Allocator;

  using iterator = general_iterator<false>;
  using const_iterator = general_iterator<true>;
  using reverse_iterator = std::reverse_iterator<iterator>;
  using const_reverse_iterator = std::reverse_iterator<const_iterator>;

  List(): fake_(BasicalNode()), list_size_(0), allocator_(NodeAllocator()) {};
  List(std::size_t number): List(number, NodeAllocator()) {};
  List(std::size_t number, const T& value): List(number, value, NodeAllocator()) {};
  List(const Allocator& allocator): fake_(BasicalNode()), list_size_(0), allocator_(allocator) {};
  List(std::size_t number, const Allocator& other_alloc);
  List(std::size_t number, const T& value, const Allocator& other_alloc);
  List(const List& other, const Allocator& other_alloc);
  List(const List& other): List(other,
       std::allocator_traits<NodeAllocator>::select_on_container_copy_construction(
                                             other.get_allocator())) {};
  ~List();

  List& operator=(const List& other);
  Allocator get_allocator() const { return allocator_; }
  std::size_t size() const { return list_size_; }
  void push_back(const T& value);
  void push_back();
  void push_front(const T& value);
  void pop_back();
  void pop_front();
  void insert(const_iterator position, const T& value);
  void erase(const_iterator position);
  void swap(List& other);

  iterator begin() { return iterator(static_cast<Node*>(fake_.next)); }
  const_iterator cbegin() const { return const_iterator(static_cast<Node*>(fake_.next)); }
  const_iterator begin() const { return cbegin(); }
  iterator end() { return ++iterator(static_cast<Node*>(fake_.previous)); }
  const_iterator cend() const { return ++const_iterator(static_cast<Node*>(fake_.previous)); }
  const_iterator end() const { return cend(); }
  reverse_iterator rbegin() { return reverse_iterator(end()); }
  const_reverse_iterator crbegin() const { return const_reverse_iterator(cend());}
  const_reverse_iterator rbegin() const { return crbegin(); }
  reverse_iterator rend() { return reverse_iterator(begin()); }
  const_reverse_iterator crend() const { return const_reverse_iterator(cbegin()); }
  const_reverse_iterator rend() const { return crend(); }
};

template <typename T, typename Allocator>
struct List<T, Allocator>::BasicalNode {
  BasicalNode* next;
  BasicalNode* previous;

  BasicalNode(): next(nullptr), previous(nullptr) {};
  ~BasicalNode() = default;
};

template <typename T, typename Allocator>
struct List<T, Allocator>::Node: BasicalNode {
  T value;

  Node(): BasicalNode() {};
  Node(const T& value): BasicalNode(), value(value) {};
  ~Node() = default;
};

template <typename T, typename Allocator>
void List<T, Allocator>::swap(List& other) {
  if (other.size() > 0 && size() > 0) {
    std::swap(fake_.next->previous, other.fake_.next->previous);
    std::swap(fake_.previous->next, other.fake_.previous->next);
  } else if (other.size() > 0 && size() == 0) {
    other.fake_.next->previous = &fake_;
    other.fake_.previous->next = &fake_;
  } else if (other.size() == 0 && size() > 0) {
    fake_.next->previous = &other.fake_;
    fake_.previous->next = &other.fake_;
  }
  std::swap(fake_.next, other.fake_.next);
  std::swap(fake_.previous, other.fake_.previous);
  std::swap(list_size_, other.list_size_);
  if (std::allocator_traits<Allocator>::propagate_on_container_swap::value) {
    std::swap(allocator_, other.allocator_);
  }
}

template <typename T, typename Allocator>
void List<T, Allocator>::swap_all(List& other) {
  swap(other);
  if (!std::allocator_traits<Allocator>::propagate_on_container_swap::value) {
    std::swap(allocator_, other.allocator_);
  }
}

template <typename T, typename Allocator>
List<T, Allocator>::List(std::size_t number, const Allocator& other_alloc):
                    fake_(BasicalNode()), list_size_(0), allocator_(other_alloc) {
  for (size_t i = 0; i < number; ++i) {
    try {
      push_back();
    } catch (...) {
      for (std::size_t j = i; j >= 1; --j) {
        pop_back();
      }
      throw;
    }
  }
}

template <typename T, typename Allocator>
List<T, Allocator>::List(std::size_t number, const T& value, const Allocator& other_alloc):
                    fake_(BasicalNode()), list_size_(0), allocator_(other_alloc) {
  for (size_t i = 0; i < number; ++i) {
    try {
      push_back(value);
    } catch (...) {
      for (std::size_t j = i; j >= 1; --j) {
        pop_back();
      }
      throw;
    }
  }
}

template <typename T, typename Allocator>
List<T, Allocator>::List(const List& other, const Allocator& other_alloc):
                    fake_(BasicalNode()), list_size_(0), allocator_(other_alloc) {
  for (auto iter = other.begin(); iter != other.end(); ++iter) {
    try {
      push_back(*iter);
    } catch (...) {
      for (auto back_iter = iter; back_iter != other.begin(); --back_iter) {
        pop_back();
      }
      throw;
    }
  }
}

template <typename T, typename Allocator>
List<T, Allocator>::~List() {
  BasicalNode* current = fake_.next;
  BasicalNode* next = nullptr;
  fake_.next = nullptr;
  while (current != nullptr && current->next != nullptr) {
    next = current->next;
    NodeTraits::destroy(allocator_, static_cast<Node*>(current));
    NodeTraits::deallocate(allocator_, static_cast<Node*>(current), 1);
    current = next;
  }
  fake_.previous = nullptr;
}

template <typename T, typename Allocator>
List<T, Allocator>& List<T, Allocator>::operator=(const List& other) {
  auto alloc = allocator_;
  if (std::allocator_traits<Allocator>::propagate_on_container_copy_assignment::value) {
    alloc = other.get_allocator();
  }
  List new_list(other, alloc);
  swap_all(new_list);
  return *this;
}

template <typename T, typename Allocator>
void List<T, Allocator>::push_back(const T& value) {
  Node* new_node = NodeTraits::allocate(allocator_, 1);
  try {
    NodeTraits::construct(allocator_, new_node, value);
  } catch (...) {
    NodeTraits::deallocate(allocator_, new_node, 1);
    throw;
  }

  if (list_size_ == 0) {
    fake_.next = new_node;
    fake_.previous = new_node;
    new_node->next = &fake_;
    new_node->previous = &fake_;
    ++list_size_;
    return;
  }
  fake_.previous->next = new_node;
  new_node->previous = fake_.previous;
  new_node->next = &fake_;
  fake_.previous = new_node;
  ++list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::push_back() {
  Node* new_node = NodeTraits::allocate(allocator_, 1);
  try {
    NodeTraits::construct(allocator_, new_node);
  } catch (...) {
    NodeTraits::deallocate(allocator_, new_node, 1);
    throw;
  }

  if (list_size_ == 0) {
    fake_.next = new_node;
    fake_.previous = new_node;
    new_node->next = &fake_;
    new_node->previous = &fake_;
    ++list_size_;
    return;
  }
  fake_.previous->next = new_node;
  new_node->previous = fake_.previous;
  new_node->next = &fake_;
  fake_.previous = new_node;
  ++list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::push_front(const T& value) {
  Node* new_node = NodeTraits::allocate(allocator_, 1);
  try {
    NodeTraits::construct(allocator_, new_node, value);
  } catch (...) {
    NodeTraits::deallocate(allocator_, new_node, 1);
    throw;
  }

  if (list_size_ == 0) {
    fake_.next = new_node;
    fake_.previous = new_node;
    new_node->next = &fake_;
    new_node->previous = &fake_;
    ++list_size_;
    return;
  }
  fake_.next->previous = new_node;
  new_node->next = fake_.next;
  new_node->previous = &fake_;
  fake_.next = new_node;
  ++list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::pop_back() {
  Node* to_delete = static_cast<Node*>(fake_.previous);
  if (list_size_ == 1) {
    fake_.next = nullptr;
    fake_.previous = nullptr;
    NodeTraits::destroy(allocator_, to_delete);
    NodeTraits::deallocate(allocator_, to_delete, 1);
    --list_size_;
    return;
  }
  to_delete->previous->next = &fake_;
  fake_.previous = to_delete->previous;
  NodeTraits::destroy(allocator_, to_delete);
  NodeTraits::deallocate(allocator_, static_cast<Node*>(to_delete), 1);
  --list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::pop_front() {
  Node* to_delete = static_cast<Node*>(fake_.next);
  if (list_size_ == 1) {
    fake_.next = nullptr;
    fake_.previous = nullptr;
    NodeTraits::destroy(allocator_, to_delete);
    NodeTraits::deallocate(allocator_, to_delete, 1);
    --list_size_;
    return;
  }
  to_delete->next->previous = &fake_;
  fake_.next = to_delete->next;
  NodeTraits::destroy(allocator_, to_delete);
  NodeTraits::deallocate(allocator_, static_cast<Node*>(to_delete), 1);
  --list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::insert(const_iterator position, const T& value) {
  Node* new_node = NodeTraits::allocate(allocator_, 1);
  try {
    NodeTraits::construct(allocator_, new_node, value);
  } catch (...) {
    NodeTraits::deallocate(allocator_, new_node, 1);
    throw;
  }

  if (list_size_ == 0) {
    fake_.next = new_node;
    fake_.previous = new_node;
    new_node->next = &fake_;
    new_node->previous = &fake_;
    ++list_size_;
    return;
  }
  Node* current = position.Pointer();
  current->previous->next = new_node;
  new_node->next = current;
  new_node->previous = current->previous;
  current->previous = new_node;
  ++list_size_;
}

template <typename T, typename Allocator>
void List<T, Allocator>::erase(const_iterator position) {
  if (list_size_ == 1) {
    fake_.next = nullptr;
    fake_.previous = nullptr;
    NodeTraits::destroy(allocator_, position.Pointer());
    NodeTraits::deallocate(allocator_, static_cast<Node*>(position.Pointer()), 1);
    --list_size_;
    return;
  }
  Node* current = position.Pointer();
  current->previous->next = current->next;
  current->next->previous = current->previous;
  NodeTraits::destroy(allocator_, current);
  NodeTraits::deallocate(allocator_, static_cast<Node*>(current), 1);
  --list_size_;
}

template <typename T, typename Allocator>
template <bool is_const>
struct List<T, Allocator>::general_iterator {
private:
  Node* node_pointer_;

public:
  using difference_type = int;
  using value_type = T;
  using reference = typename std::conditional<is_const, const T&, T&>::type;
  using pointer = typename std::conditional<is_const, const T*, T*>::type;
  using iterator_category = std::bidirectional_iterator_tag;

  general_iterator(): node_pointer_(nullptr) {};
  general_iterator(Node* node_pointer_): node_pointer_(node_pointer_) {};

  reference operator*() const {
    return node_pointer_->value;
  }

  pointer operator->() const {
    return &node_pointer_->value;
  }

  general_iterator& operator++() {
    node_pointer_ = static_cast<Node*>(node_pointer_->next);
    return *this;
  }

  general_iterator operator++(int) {
    general_iterator new_iter(*this);
    ++(*this);
    return new_iter;
  }

  general_iterator& operator--() {
    node_pointer_ = static_cast<Node*>(node_pointer_->previous);
    return *this;
  }

  general_iterator operator--(int) {
    general_iterator new_iter(*this);
    --(*this);
    return new_iter;
  }

  bool operator==(const_iterator other) const {
    return node_pointer_ == other.Pointer();
  }

  bool operator!=(const_iterator other) const {
    return !(*this == other);
  }

  operator const_iterator() const {
    return const_iterator(node_pointer_);
  }

  Node* Pointer() const {
    return node_pointer_;
  }
};