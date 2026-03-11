#include <memory>
#include <vector>
#include <algorithm>
#include <utility>
#include <stdexcept>
#include <iostream>

template <typename T, typename Allocator = std::allocator<T>>
class List {
private:
  struct BasicalNode;
  struct Node;
  template <bool is_const>
  struct general_iterator;

  using NodeAllocator = typename std::allocator_traits<Allocator>::template rebind_alloc<Node>;
  using NodeTraits = std::allocator_traits<NodeAllocator>;
  using allocator_type = Allocator;
  using iterator = general_iterator<false>;
  using const_iterator = general_iterator<true>;
  using reverse_iterator = std::reverse_iterator<iterator>;
  using const_reverse_iterator = std::reverse_iterator<const_iterator>;

  template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
  friend class UnorderedMap;

  List() = default;
  explicit List(const Allocator& allocator_):
           fake_(BasicalNode()), list_size_(0), allocator_(allocator_) {};
  List(const List& other, const Allocator& other_alloc);
  List(const List& other): List(other, std::allocator_traits<NodeAllocator>::
                                select_on_container_copy_construction(other.get_allocator())) {};
  List(List&& other, const Allocator& other_alloc);
  List(List&& other): List(std::move(other), other.get_allocator()) {};
  ~List();

  List& operator=(const List& other);
  List& operator=(List&& other);
  Allocator get_allocator() const { return allocator_; }
  std::size_t size() const { return list_size_; }
  template <typename P>
  void universal_push_back(P&& value);
  void push_back(const T& value) { universal_push_back(value); }
  void push_back(T&& value) { universal_push_back(std::move(value)); }
  void push_back(BasicalNode* node);
  void pop_back();
  void insert(BasicalNode* position, BasicalNode* node);
  void insert(BasicalNode* position, T&& node) { emplace(position, std::move(node)); }
  template<class... Args>
  void emplace(BasicalNode* position, Args&&... args);
  void erase(BasicalNode* position);
  void erase_without_delete(BasicalNode* position);
  void swap(List& other);

  iterator begin() { return iterator(static_cast<Node*>(fake_.next)); }
  const_iterator cbegin() const { return const_iterator(static_cast<Node*>(fake_.next)); }
  const_iterator begin() const { return cbegin(); }
  iterator end() { return ++iterator(static_cast<Node*>(fake_.previous)); }
  const_iterator cend() const { return ++const_iterator(static_cast<Node*>(fake_.previous)); }
  const_iterator end() const { return cend(); }

  BasicalNode fake_;
  std::size_t list_size_;
  [[no_unique_address]] NodeAllocator allocator_;
};

template <typename Key, typename Value, typename Hash = std::hash<Key>,
          typename Equal = std::equal_to<Key>,
          typename Alloc = std::allocator<std::pair<const Key, Value>>>
class UnorderedMap {
public:
  using NodeType = typename std::pair<const Key,Value>;
  using key_type = Key;
  using mapped_type = Value;
  using value_type = NodeType;
  using hasher = Hash;
  using key_equal = Equal;
  using allocator_type = Alloc;
  using reference = NodeType&;
  using const_reference = const NodeType&;
  using pointer =typename std::allocator_traits<Alloc>::pointer;
  using const_pointer = typename std::allocator_traits<Alloc>::const_pointer;

  template <bool is_const>
  struct general_iterator;
  using iterator = general_iterator<false>;
  using const_iterator = general_iterator<true>;

  UnorderedMap(): list_(allocator_) {}
  UnorderedMap(const UnorderedMap& other, Alloc allocator);
  UnorderedMap(const UnorderedMap& other):
     UnorderedMap(other, std::allocator_traits<Alloc>::
                  select_on_container_copy_construction(other.allocator_)) {}
  UnorderedMap(UnorderedMap&& other);
  UnorderedMap(UnorderedMap&& other, Alloc allocator);
  UnorderedMap& operator=(const UnorderedMap& other);
  UnorderedMap& operator=(UnorderedMap&& other);

  void rehash(size_t count);
  void reserve(size_t count) { rehash(count); }
  Value& operator[](const Key& key);
  Value& operator[](Key&& key);
  Value& at(const Key& key);
  const Value& at(const Key& key) const;
  size_t size() const { return value_size_; }
  std::pair<iterator, bool> insert(const NodeType& key_value) { return emplace(key_value); }
  template <typename P>
  std::pair<iterator, bool> insert(P&& key_value) { return emplace(std::forward<P>(key_value)); }
  template <typename InputIter>
  void insert(InputIter start, InputIter finish);
  template< class... Args>
  std::pair<iterator, bool> emplace(Args&&... args);
  void erase(iterator position);
  void erase(iterator start, iterator finish);
  iterator find(const Key& key);
  const_iterator find(const Key& key) const;
  double load_factor() const {
    return static_cast<double>(value_size_) / static_cast<double>(table_.size());
  }
  double max_load_factor() const { return MY_MAX_LOAD_FACTOR_; }
  void max_load_factor(double mlf) { MY_MAX_LOAD_FACTOR_ = mlf; }
  void swap(UnorderedMap& other);

  iterator begin() { return iterator(value_size_ == 0 ? &list_.fake_ : list_.fake_.next); }
  const_iterator cbegin() const {
    return const_iterator(value_size_ == 0 ? &list_.fake_ : list_.fake_.next);
  }
  const_iterator begin() const { return cbegin(); }
  iterator end() { return iterator(&list_.fake_); }
  const_iterator cend() const { return const_iterator(&list_.fake_); }
  const_iterator end() const { return cend(); }

private:
  struct ListType;
  using ListBasicalNode = typename List<ListType, Alloc>::BasicalNode;
  using ListNode = typename List<ListType, Alloc>::Node;

  const double ACCURACY_ = 1e-9;

  [[no_unique_address]] Alloc allocator_;
  List<ListType, Alloc> list_;
  std::vector<ListBasicalNode*> table_;
  size_t value_size_ = 0;
  double MY_MAX_LOAD_FACTOR_ = 1;
  [[no_unique_address]] Hash hash_;
  [[no_unique_address]] Equal equal_;
};

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
struct UnorderedMap<Key, Value, Hash, Equal, Alloc>::ListType {
  ListType(const ListType& other) = default;

  ListType(ListType&& other): key_value(other.key_value), hash(other.hash) {
    other.key_value = nullptr;
    other.hash = 0;
  }

  ListType(const NodeType& key_value_meaning, size_t hash): key_value(nullptr), hash(hash) {
    auto node_type_alloc = Alloc();
    key_value = std::allocator_traits<Alloc>::allocate(node_type_alloc, 1);
    std::allocator_traits<Alloc>::construct(node_type_alloc, key_value, key_value_meaning);
  }

  template <typename... Args>
  ListType(size_t hash, Args&&... args): key_value(nullptr), hash(hash) {
    auto node_type_alloc = Alloc();
    key_value = std::allocator_traits<Alloc>::allocate(node_type_alloc, 1);
    std::allocator_traits<Alloc>::construct(node_type_alloc, key_value, std::forward<Args>(args)...);
  }

  ~ListType() {
    if (key_value != nullptr) {
      auto node_type_alloc = Alloc();
      std::allocator_traits<Alloc>::destroy(node_type_alloc, key_value);
      std::allocator_traits<Alloc>::deallocate(node_type_alloc, key_value, 1);
    }
  }
        
  NodeType* key_value;
  size_t hash;
};

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
void UnorderedMap<Key, Value, Hash, Equal, Alloc>::rehash(size_t count) {
  if (table_.size() >= count) {
    return;
  }
  std::vector<ListBasicalNode*> new_table(count);
  List<ListType, Alloc> new_list(allocator_);
  ListBasicalNode* current_node = begin().node_pointer_;
  ListBasicalNode* next_node = nullptr;

  for (size_t i = 0; i < value_size_; ++i) {
    next_node = current_node->next;
    size_t hash_value = static_cast<ListNode*>(current_node)->value.hash % new_table.size();
    list_.erase_without_delete(current_node);
    if (new_table[hash_value] == nullptr) {
      new_list.push_back(current_node);
      new_table[hash_value] = (--new_list.end()).Pointer();
    } else {
      new_list.insert(new_table[hash_value], current_node);
      new_table[hash_value] = new_table[hash_value]->previous;
    }
    current_node = next_node;
  }
  list_ = std::move(new_list);
  table_ = std::move(new_table);
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
UnorderedMap<Key, Value, Hash, Equal, Alloc>::UnorderedMap(const UnorderedMap& other, Alloc allocator):
             allocator_(allocator), list_(allocator_), table_(other.table_.size()) {

  for (auto iter = other.begin(); iter != other.end(); ++iter) {
    NodeType key_value = std::make_pair((*iter).first, (*iter).second);
    size_t key_hash = iter.hash();
    size_t hash_value = key_hash % table_.size();
    if (table_[hash_value] == nullptr) {
      list_.push_back(ListType(key_value, key_hash));
      table_[hash_value] = (--list_.end()).Pointer();
    } else {
      list_.insert(table_[hash_value], ListType(key_value, key_hash));
      table_[hash_value] = table_[hash_value]->previous;
    }
  }
  value_size_ = other.value_size_;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
UnorderedMap<Key, Value, Hash, Equal, Alloc>::UnorderedMap(UnorderedMap&& other, Alloc allocator):
             allocator_(allocator), list_(allocator_), table_(other.table_.size()) {

  for (auto iter = other.begin(); iter != other.end(); ++iter) {
    size_t key_hash = iter.hash();
    size_t hash_value = key_hash % table_.size();
    if (table_[hash_value] == nullptr) {
      list_.emplace(list_.end().Pointer(), key_hash, (*iter).first, std::move((*iter).second));
      table_[hash_value] = (--list_.end()).Pointer();
    } else {
      list_.emplace(table_[hash_value], key_hash, (*iter).first, std::move((*iter).second));
      table_[hash_value] = table_[hash_value]->previous;
    }
  }
  value_size_ = other.value_size_;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
UnorderedMap<Key, Value, Hash, Equal, Alloc>::UnorderedMap(UnorderedMap&& other):
    allocator_(std::move(other.allocator_)), list_(std::move(other.list_)),
    table_(std::move(other.table_)), value_size_(other.value_size_) {
  other.value_size_ = 0;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
UnorderedMap<Key, Value, Hash, Equal, Alloc>&
UnorderedMap<Key, Value, Hash, Equal, Alloc>::operator=(const UnorderedMap& other) {
  if (std::allocator_traits<Alloc>::propagate_on_container_copy_assignment::value) {
    allocator_ = other.allocator_;
  }
  UnorderedMap new_map(other, allocator_);

  list_.swap(new_map.list_);
  std::swap(table_, new_map.table_);
  std::swap(value_size_, new_map.value_size_);
  return *this;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
UnorderedMap<Key, Value, Hash, Equal, Alloc>&
UnorderedMap<Key, Value, Hash, Equal, Alloc>::operator=(UnorderedMap&& other) {
  if constexpr (!std::allocator_traits<Alloc>::propagate_on_container_move_assignment::value and
                allocator_ != other.allocator_) {
    UnorderedMap new_map(std::move(other), allocator_);
    list_.swap(new_map.list_);
    std::swap(table_, new_map.table_);
    std::swap(value_size_, new_map.value_size_);
  } else {
    UnorderedMap new_map(std::move(other));
    if (std::allocator_traits<Alloc>::propagate_on_container_move_assignment::value) {
      allocator_ = other.allocator_;
    }

    list_.swap(new_map.list_);
    std::swap(table_, new_map.table_);
    std::swap(value_size_, new_map.value_size_);
  }
  return *this;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
Value& UnorderedMap<Key, Value, Hash, Equal, Alloc>::operator[](const Key& key) {
  auto iter = find(key);
  if (iter == end()) {
    iter = emplace(key, Value()).first;
  }
  return (*iter).second;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
Value& UnorderedMap<Key, Value, Hash, Equal, Alloc>::operator[](Key&& key) {
  return (emplace(std::move(key), Value()).first)->second;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
Value& UnorderedMap<Key, Value, Hash, Equal, Alloc>::at(const Key& key) {
  auto iter = find(key);
  if (iter == end()) {
    throw std::out_of_range("at()");
  }
  return (*iter).second;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
const Value& UnorderedMap<Key, Value, Hash, Equal, Alloc>::at(const Key& key) const {
  auto iter = find(key);
  if (iter == end()) {
    throw std::out_of_range("at()");
  }
  return (*iter).second;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
template <typename InputIter>
void UnorderedMap<Key, Value, Hash, Equal, Alloc>::insert(InputIter start, InputIter finish) {
  if (MY_MAX_LOAD_FACTOR_ <= ACCURACY_ + static_cast<double>(value_size_ + 1) /
                                         static_cast<double>(table_.size())) {
    rehash((table_.size() + 1) * 3);
  }
  for (auto iter = start; iter != finish; ++iter) {
    insert(*iter);
  }
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
template< class... Args>
std::pair<typename UnorderedMap<Key, Value, Hash, Equal, Alloc>::iterator, bool>
UnorderedMap<Key, Value, Hash, Equal, Alloc>::emplace(Args&&... args) {
  if (MY_MAX_LOAD_FACTOR_ - static_cast<double>(value_size_ + 1) /
                            static_cast<double>(table_.size()) <= ACCURACY_) {
    rehash((table_.size() + 1) * 3);
  }

  bool increased = false;
  if (table_.size() == 0) {
    rehash((table_.size() + 1) * 3);
    increased = true;
  }

  list_.emplace(list_.begin().Pointer(), 0, std::forward<Args>(args)...);
  ListNode* node = static_cast<ListNode*>((list_.begin()).Pointer());
  list_.erase_without_delete(node);

  size_t key_hash = hash_(node->value.key_value->first);
  node->value.hash = key_hash;
  size_t hash_value = key_hash % table_.size();

  if (table_[hash_value] == nullptr) {
    try {
      list_.push_back(static_cast<ListBasicalNode*>(node));
    } catch (...) {
      if (increased) {
        table_ = {};
      }
      throw;
    }
    ++value_size_;
    table_[hash_value] = (--list_.end()).Pointer();
    return {iterator(table_[hash_value]), true};
  }

  for (auto ptr = table_[hash_value]; ptr->next != list_.fake_.next and
       static_cast<ListNode*>(ptr)->value.hash % table_.size() == hash_value; ptr = ptr->next) {
    try {
      if (equal_(static_cast<ListNode*>(ptr)->value.key_value->first, node->value.key_value->first)) {
        using ListNodeAlloc = typename std::allocator_traits<Alloc>::template rebind_alloc<ListNode>;
        auto list_node_alloc = static_cast<ListNodeAlloc>(allocator_);
        std::allocator_traits<ListNodeAlloc>::destroy(list_node_alloc, node);
        std::allocator_traits<ListNodeAlloc>::deallocate(list_node_alloc, node, 1);
        return {iterator(ptr), false};
      }
    } catch (...) {
      if (increased) {
        table_ = {};
      }
      throw;
    }
  }

  list_.insert(table_[hash_value], node);
  ++value_size_;
  table_[hash_value] = table_[hash_value]->previous;
  return {iterator(table_[key_hash % table_.size()]), true};
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
void UnorderedMap<Key, Value, Hash, Equal, Alloc>::erase(iterator position) {
  ListBasicalNode* current =  const_cast<ListBasicalNode*>(position.node_pointer_);

  if (table_[static_cast<ListNode*>(current)->value.hash % table_.size()] == current) {
    if (current->next->next != list_.fake_.next and
        static_cast<ListNode*>(current->next)->value.hash ==
        static_cast<ListNode*>(current)->value.hash) {
      table_[static_cast<ListNode*>(current)->value.hash % table_.size()] = current->next;
    } else {
      table_[static_cast<ListNode*>(current)->value.hash % table_.size()] = nullptr;
    }
  }
  list_.erase(current);
  --value_size_;
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
void UnorderedMap<Key, Value, Hash, Equal, Alloc>::erase(iterator start, iterator finish) {
  iterator next = iterator(start.node_pointer_->next);
  while (start != finish) {
    erase(start);
    start = next;
    next = iterator(start.node_pointer_->next);
  }
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
typename UnorderedMap<Key, Value, Hash, Equal, Alloc>::iterator
UnorderedMap<Key, Value, Hash, Equal, Alloc>::find(const Key& key) {
  if (table_.size() == 0) {
    return end();
  }
  size_t hash_value = hash_(key) % table_.size();
  if (table_[hash_value] == nullptr) {
    return end();
  }
  for (ListBasicalNode* ptr = table_[hash_value]; ptr->next != list_.fake_.next
       and static_cast<ListNode*>(ptr)->value.hash % table_.size() == hash_value; ptr = ptr->next) {
    if (equal_(static_cast<ListNode*>(ptr)->value.key_value->first, key)) {
      return iterator(ptr);
    }
  }
  return end();
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
typename UnorderedMap<Key, Value, Hash, Equal, Alloc>::const_iterator
UnorderedMap<Key, Value, Hash, Equal, Alloc>::find(const Key& key) const {
  return static_cast<const_iterator>(find(key));
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
void UnorderedMap<Key, Value, Hash, Equal, Alloc>::swap(UnorderedMap& other) {
  list_.swap(other.list_);
  std::swap(table_, other.table_);
  std::swap(value_size_, other.value_size_);
  std::swap(hash_, other.hash_);
  std::swap(equal_, other.equal_);
  if (std::allocator_traits<Alloc>::propagate_on_container_swap::value) {
    std::swap(allocator_, other.allocator_);
  }
}

template <typename Key, typename Value, typename Hash, typename Equal, typename Alloc>
template <bool is_const>
struct UnorderedMap<Key, Value, Hash, Equal, Alloc>::general_iterator {
private:
  using ListNodePtr = typename std::conditional_t<is_const, const ListNode*, ListNode*>;
  using ListBaseNodePtr = typename std::conditional_t<is_const, const ListBasicalNode*,
                                                      ListBasicalNode*>;

  template <typename, typename, typename, typename, typename>
  friend class UnorderedMap;
  template <bool> 
  friend struct general_iterator;

public:
  using difference_type = int;
  using value_type = NodeType;
  using reference = typename std::conditional_t<is_const, const NodeType&, NodeType&>;
  using pointer = typename std::conditional_t<is_const, const NodeType*, NodeType*>;
  using iterator_category = std::forward_iterator_tag;

  general_iterator(): node_pointer_(nullptr) {}
  explicit general_iterator(ListBaseNodePtr node_pointer): node_pointer_(node_pointer) {};

  reference operator*() const {
    return *(static_cast<ListNodePtr>(node_pointer_)->value.key_value);
  }

  pointer operator->() const {
    return (static_cast<ListNodePtr>(node_pointer_))->value.key_value;
  }

  general_iterator& operator++() {
    node_pointer_ = node_pointer_->next;
    return *this;
  }

  general_iterator operator++(int) {
    general_iterator new_iter(*this);
    ++(*this);
    return new_iter;
  }

  bool operator==(const_iterator other) const {
    return node_pointer_ == other.node_pointer_;
  }

  bool operator!=(const_iterator other) const {
    return not(*this == other);
  }

  size_t hash() const {
    return (static_cast<ListNodePtr>(node_pointer_))->value.hash;
  }

  operator const_iterator() const {
    return const_iterator(node_pointer_);
  }

private:
  ListBaseNodePtr node_pointer_;
};

template <typename T, typename Allocator>
struct List<T, Allocator>::BasicalNode {
  BasicalNode* next;
  BasicalNode* previous;

  BasicalNode(): next(nullptr), previous(nullptr) {};
};

template <typename T, typename Allocator>
struct List<T, Allocator>::Node: BasicalNode {
  T value;

  Node() = default;
  explicit Node(const T& value): BasicalNode(), value(value) {};
  explicit Node(T&& value): BasicalNode(), value(std::move(value)) {};
  template <typename... Args>
  explicit Node(Args&&... args): BasicalNode(), value(std::forward<Args>(args)...) {};
};

template <typename T, typename Allocator>
void List<T, Allocator>::swap(List& other) {
  if (other.size() > 0 and size() > 0) {
    std::swap(fake_.next->previous, other.fake_.next->previous);
    std::swap(fake_.previous->next, other.fake_.previous->next);
  } else if (other.size() > 0 and size() == 0) {
    other.fake_.next->previous = &fake_;
    other.fake_.previous->next = &fake_;
  } else if (other.size() == 0 and size() > 0) {
    fake_.next->previous = &other.fake_;
    fake_.previous->next = &other.fake_;
  }
  std::swap(fake_.next, other.fake_.next);
  std::swap(fake_.previous, other.fake_.previous);
  std::swap(list_size_, other.list_size_);
  std::swap(allocator_, other.allocator_);
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
List<T, Allocator>::List(List&& other, const Allocator& other_alloc):
                         list_size_(other.size()), allocator_(other_alloc) {
  if (other.list_size_ != 0) {
    other.fake_.next->previous = &fake_;
    other.fake_.previous->next = &fake_;
  }
  fake_.next = other.fake_.next;
  fake_.previous = other.fake_.previous;
  other.fake_.previous = nullptr;
  other.fake_.next = nullptr;
  other.list_size_ = 0;
}

template <typename T, typename Allocator>
List<T, Allocator>::~List() {
  BasicalNode* current = fake_.next;
  BasicalNode* next = nullptr;
  fake_.next = nullptr;
  while (current != nullptr and current->next != nullptr) {
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
  swap(new_list);
  return *this;
}

template <typename T, typename Allocator>
List<T, Allocator>& List<T, Allocator>::operator=(List&& other) {
  auto alloc = allocator_;
  if (std::allocator_traits<Allocator>::propagate_on_container_move_assignment::value) {
    alloc = other.get_allocator();
  }
  List new_list(std::move(other), alloc);
  swap(new_list);
  return *this;
}

template <typename T, typename Allocator>
template <typename P>
void List<T, Allocator>::universal_push_back(P&& value) {
  emplace(&fake_, std::forward<P>(value));
}

template <typename T, typename Allocator>
void List<T, Allocator>::push_back(BasicalNode* node) {
  if (list_size_ == 0) {
    fake_.next = node;
    fake_.previous = node;
    node->next = &fake_;
    node->previous = &fake_;
    ++list_size_;
    return;
  }
  fake_.previous->next = node;
  node->previous = fake_.previous;
  node->next = &fake_;
  fake_.previous = node;
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
void List<T, Allocator>::insert(BasicalNode* position, BasicalNode* node) {
  if (list_size_ == 0) {
    fake_.next = node;
    fake_.previous = node;
    node->next = &fake_;
    node->previous = &fake_;
    ++list_size_;
    return;
  } 
  position->previous->next = node;
  node->next = position;
  node->previous = position->previous;
  position->previous = node;
  ++list_size_;
}

template <typename T, typename Allocator>
template <typename... Args>
void List<T, Allocator>::emplace(BasicalNode* position, Args&&... args) {
  Node* new_node = NodeTraits::allocate(allocator_, 1);
  try {
    NodeTraits::construct(allocator_, new_node, std::forward<Args>(args)...);
  } catch (...) {
    NodeTraits::deallocate(allocator_, new_node, 1);
    throw;
  }
  insert(position, new_node);
}

template <typename T, typename Allocator>
void List<T, Allocator>::erase(BasicalNode* position) {
  erase_without_delete(position);
  NodeTraits::destroy(allocator_, static_cast<Node*>(position));
  NodeTraits::deallocate(allocator_, static_cast<Node*>(position), 1);
}

template <typename T, typename Allocator>
void List<T, Allocator>::erase_without_delete(BasicalNode* position) {
  if (list_size_ == 1) {
    fake_.next = nullptr;
    fake_.previous = nullptr;
    --list_size_;
    return;
  }
  position->previous->next = position->next;
  position->next->previous = position->previous;
  --list_size_;
}

template <typename T, typename Allocator>
template <bool is_const>
struct List<T, Allocator>::general_iterator {
public:
  using difference_type = int;
  using value_type = T;
  using reference = typename std::conditional_t<is_const, const T&, T&>;
  using pointer = typename std::conditional_t<is_const, const T*, T*>;
  using iterator_category = std::bidirectional_iterator_tag;

  general_iterator(): node_pointer_(nullptr) {}
  general_iterator(Node* node_pointer): node_pointer_(node_pointer) {};

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
    return not(*this == other);
  }

  operator const_iterator() const {
    return const_iterator(node_pointer_);
  }

  Node* Pointer() const {
    return node_pointer_;
  }

private:
  Node* node_pointer_;
};