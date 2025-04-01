#include <memory>
#include <iostream>

template <typename T>
class EnableSharedFromThis;

struct BaseControlBlock {
  size_t shared_ptr_number = 0;
  size_t weak_ptr_number = 0;
  virtual void* get_object() = 0;
  virtual void destroy_object() = 0;
  virtual void destroy_left_part() = 0;
  virtual void destroy_control_block() = 0;
  virtual ~BaseControlBlock() = default;
};

template <typename Y, typename Deleter, typename Allocator>
struct DefaultControlBlock: BaseControlBlock {
  using ControlBlockAllocator = typename std::allocator_traits<Allocator>::
                                template rebind_alloc<DefaultControlBlock<Y, Deleter, Allocator>>;
  using ControlBlockTraits = std::allocator_traits<ControlBlockAllocator>;

  Y* object;
  Deleter deleter;
  Allocator allocator;
  void* get_object() override { return object; }
  void destroy_object() override {
    deleter(object);
  }
  void destroy_left_part() override {
    auto concrete_block_allocator = static_cast<ControlBlockAllocator>(allocator);
    this->~DefaultControlBlock();
    ControlBlockTraits::deallocate(concrete_block_allocator, this, 1);
  }
  void destroy_control_block() override {
    destroy_object();
    destroy_left_part();
  }
  DefaultControlBlock(Y* object): object(object), deleter(std::default_delete<Y>()),
                                  allocator(std::allocator<Y>()) {}
  DefaultControlBlock(Y* object, Deleter deleter, Allocator allocator):
                      object(object), deleter(deleter), allocator(allocator) {}
  ~DefaultControlBlock() override = default;
};

template <typename Y, typename Allocator> 
struct FunctionalControlBlock: BaseControlBlock {
  using ObjectAllocator = typename std::allocator_traits<Allocator>::template rebind_alloc<Y>;
  using ObjectTraits = std::allocator_traits<ObjectAllocator>;
  using ControlBlockAllocator = typename std::allocator_traits<Allocator>::
                                template rebind_alloc<FunctionalControlBlock<Y, Allocator>>;
  using ControlBlockTraits = std::allocator_traits<ControlBlockAllocator>;

  Y object;
  Allocator allocator;
  void* get_object() override {
    return &object;
  }
  void destroy_object() override {
    ObjectTraits::destroy(allocator, &object);
  }
  void destroy_left_part() override {
    auto concrete_block_allocator = static_cast<ControlBlockAllocator>(allocator);
    ControlBlockTraits::deallocate(concrete_block_allocator, this, 1);
  }
  void destroy_control_block() override {
    auto concrete_block_allocator = static_cast<ControlBlockAllocator>(allocator);
    ControlBlockTraits::destroy(concrete_block_allocator, this);
    ControlBlockTraits::deallocate(concrete_block_allocator, this, 1);
  }
  template <typename ...Args>
  FunctionalControlBlock(Allocator allocator, Args&& ...args): object(std::forward<Args>(args)...),
                                                               allocator(allocator) {}
  FunctionalControlBlock(FunctionalControlBlock&& other): object(std::move(other.object)),
                                                          allocator(other.allocator) {}
  ~FunctionalControlBlock() override = default;
};

template <typename T>
class SharedPtr {
private:
  T* aliase_;
  BaseControlBlock* data_;

  SharedPtr(bool, T* aliase, BaseControlBlock* data): aliase_(aliase), data_(data) {}
  void delete_ptrs();
  BaseControlBlock* data_ptr() const { return data_; }
  
  template <typename>
  friend class SharedPtr;
  template <typename>
  friend class WeakPtr;
  template <typename Y, typename Allocator, typename... Args>
  friend SharedPtr<Y> allocateShared(const Allocator& allocator, Args&&... args);
  template <typename Y, typename... Args>
  friend SharedPtr<Y> makeShared(Args&&... args);

public:
  SharedPtr();
  template <typename Y, typename Deleter, typename Allocator>
  SharedPtr(Y* object, Deleter deleter, Allocator allocator);
  template <typename Y, typename Deleter>
  SharedPtr(Y* object, Deleter deleter): SharedPtr(object, deleter, std::allocator<Y>()) {}
  template <typename Y>
  SharedPtr(Y* object): SharedPtr(object, std::default_delete<Y>(), std::allocator<Y>()) {}
  template <typename Y>
  SharedPtr(const SharedPtr<Y>& other, T* aliase_);
  template <typename Y>
  SharedPtr(const SharedPtr<Y>& other): SharedPtr(other, nullptr) {}
  SharedPtr(const SharedPtr& other);
  template <typename Y>
  SharedPtr(SharedPtr<Y>&& other);
  SharedPtr(SharedPtr&& other);
  ~SharedPtr();

  template <typename Y>
  SharedPtr& operator=(const SharedPtr<Y>& other);
  SharedPtr& operator=(const SharedPtr& other);
  template <typename Y>
  SharedPtr& operator=(SharedPtr<Y>&& other);
  SharedPtr& operator=(SharedPtr&& other);

  size_t use_count() const { return data_->shared_ptr_number; }
  template <typename Y>
  void reset(Y* other_object);
  void reset();
  T* get() const;
  T& operator*() const { return *get(); }
  T* operator->() const { return get(); }
  void swap(SharedPtr& other);
  BaseControlBlock* get_data() const { return data_; }
};

template <typename T>
SharedPtr<T>::SharedPtr() {
  auto new_data = new DefaultControlBlock<T, std::default_delete<T>, std::allocator<T>>(nullptr);
  aliase_ = new_data->object;
  data_ = new_data;
  ++data_->shared_ptr_number;
}

template <typename T>
template <typename Y, typename Deleter, typename Allocator>
SharedPtr<T>::SharedPtr(Y* object, Deleter deleter, Allocator allocator) {
  if constexpr (std::is_base_of_v<EnableSharedFromThis<Y>, Y>) {
    object->weak_ptr_ = *this;
  }

  using ControlBlockAllocator = typename std::allocator_traits<Allocator>::
                                template rebind_alloc<DefaultControlBlock<Y, Deleter, Allocator>>;
  using ControlBlockTraits = std::allocator_traits<ControlBlockAllocator>;
  auto concrete_block_allocator = static_cast<ControlBlockAllocator>(allocator);
  auto new_data = ControlBlockTraits::allocate(concrete_block_allocator, 1);
  new(new_data) DefaultControlBlock<Y, Deleter, Allocator>(object, deleter, allocator);
  aliase_ = new_data->object;
  data_ = new_data;
  ++data_->shared_ptr_number;
}

template <typename T>
template <typename Y>
SharedPtr<T>::SharedPtr(const SharedPtr<Y>& other, T* aliase_): aliase_(aliase_),
                                                                data_(other.data_ptr()) {
  ++(data_->shared_ptr_number);
}

template <typename T>
SharedPtr<T>::SharedPtr(const SharedPtr& other): aliase_(other.aliase_), data_(other.data_) {
  ++(data_->shared_ptr_number);
}

template <typename T>
template <typename Y>
SharedPtr<T>::SharedPtr(SharedPtr<Y>&& other): aliase_(other.aliase_), data_(other.data_ptr()) {
  other.aliase_ = nullptr;
  other.data_ = nullptr;
}

template <typename T>
SharedPtr<T>::SharedPtr(SharedPtr&& other): aliase_(other.aliase_), data_(other.data_) {
  other.aliase_ = nullptr;
  other.data_ = nullptr;
}

template <typename T>
void SharedPtr<T>::delete_ptrs() {
  if (data_->shared_ptr_number > 0) {
    return;
  }
  if (data_->weak_ptr_number == 0) {
    data_->destroy_control_block();
  } else {
    data_->destroy_object();
  }
}

template <typename T>
SharedPtr<T>::~SharedPtr() {
  if (data_ != nullptr) {
    --data_->shared_ptr_number;
    delete_ptrs();
  }
}

template <typename T>
template <typename Y>
SharedPtr<T>& SharedPtr<T>::operator=(const SharedPtr<Y>& other) {
  --(data_->shared_ptr_number);
  delete_ptrs();
  aliase_ = other.aliase_;
  data_ = other.data_;
  ++(data_->shared_ptr_number);
  return *this;
}

template <typename T>
SharedPtr<T>& SharedPtr<T>::operator=(const SharedPtr& other) {
  --(data_->shared_ptr_number);
  delete_ptrs();
  aliase_ = other.aliase_;
  data_ = other.data_;
  ++(data_->shared_ptr_number);
  return *this;
}

template <typename T>
template <typename Y>
SharedPtr<T>& SharedPtr<T>::operator=(SharedPtr<Y>&& other) {
  SharedPtr new_ptr(std::move(other));
  std::swap(aliase_, new_ptr.aliase_);
  std::swap(data_, new_ptr.data_);
  return *this;
}

template <typename T>
SharedPtr<T>& SharedPtr<T>::operator=(SharedPtr&& other) {
  SharedPtr new_ptr(std::move(other));
  std::swap(aliase_, new_ptr.aliase_);
  std::swap(data_, new_ptr.data_);
  return *this;
}

template <typename T>
template <typename Y>
void SharedPtr<T>::reset(Y* other_object) {
  *this = SharedPtr(other_object, std::default_delete<Y>(), std::allocator<Y>());
}

template <typename T>
void SharedPtr<T>::reset() {
  if (data_ != nullptr) {
    --data_->shared_ptr_number;
    delete_ptrs();
    data_ = nullptr;
  }
}

template <typename T>
T* SharedPtr<T>::get() const {
  if (data_ == nullptr) {
    return nullptr;
  }
  return aliase_;
}

template <typename T>
void SharedPtr<T>::swap(SharedPtr& other) {
  std::swap(aliase_, other.aliase_);
  std::swap(data_, other.data_);
}

template <typename T>
class WeakPtr {
private:
  BaseControlBlock* data_;

  void delete_ptrs();

  template <typename>
  friend class WeakPtr;
  template <typename Y>
  friend class EnableSharedFromThis;

public:
  WeakPtr();
  template <typename Y>
  WeakPtr(const SharedPtr<Y>& shared_ptr): data_(shared_ptr.get_data()) { ++data_->weak_ptr_number; }
  WeakPtr(const WeakPtr& other): data_(other.data_) { ++data_->weak_ptr_number; }
  template <typename Y>
  WeakPtr(const WeakPtr<Y>& other): data_(other.data_) { ++data_->weak_ptr_number; }
  WeakPtr(WeakPtr&& other): data_(other.data_) { other.data_ = nullptr; }
  template <typename Y>
  WeakPtr(WeakPtr<Y>&& other): data_(other.data_) { other.data_ = nullptr; }
  ~WeakPtr();

  template <typename Y>
  WeakPtr& operator=(const SharedPtr<Y>& shared_ptr);
  WeakPtr& operator=(const WeakPtr& other);
  template <typename Y>
  WeakPtr& operator=(const WeakPtr<Y>& other);
  WeakPtr& operator=(WeakPtr&& other);
  template <typename Y>
  WeakPtr& operator=(WeakPtr<Y>&& other);

  bool expired() const { return data_->shared_ptr_number == 0; }
  SharedPtr<T> lock() const;
  size_t use_count() const { return data_->shared_ptr_number; }
};

template <typename T>
void WeakPtr<T>::delete_ptrs() {
  if (data_->weak_ptr_number + data_->shared_ptr_number > 0) {
    return;
  }
  data_->destroy_left_part();
}

template <typename T>
WeakPtr<T>::WeakPtr() {
  auto new_data = new DefaultControlBlock<T, std::default_delete<T>, std::allocator<T>>(nullptr);
  data_ = new_data;
  ++data_->weak_ptr_number;
}

template <typename T>
WeakPtr<T>::~WeakPtr() {
  if (data_ != nullptr) {
    --data_->weak_ptr_number;
    delete_ptrs();
  }
}

template <typename T>
template <typename Y>
WeakPtr<T>& WeakPtr<T>::operator=(const SharedPtr<Y>& shared_ptr) {
  --data_->weak_ptr_number;
  delete_ptrs();
  data_ = shared_ptr.get_data();
  ++data_->weak_ptr_number;
  return *this;
}

template <typename T>
WeakPtr<T>& WeakPtr<T>::operator=(const WeakPtr& other) {
  --data_->weak_ptr_number;
  delete_ptrs();
  data_ = other.data_;
  ++data_->weak_ptr_number;
  return *this;
}

template <typename T>
template <typename Y>
WeakPtr<T>& WeakPtr<T>::operator=(const WeakPtr<Y>& other) {
  --data_->weak_ptr_number;
  delete_ptrs();
  data_ = other.data_;
  ++data_->weak_ptr_number;
  return *this;
}

template <typename T>
WeakPtr<T>& WeakPtr<T>::operator=(WeakPtr&& other) {
  WeakPtr new_ptr(std::move(other));
  std::swap(data_, new_ptr.data_);
  return *this;
}

template <typename T>
template <typename Y>
WeakPtr<T>& WeakPtr<T>::operator=(WeakPtr<Y>&& other) {
  WeakPtr new_ptr(std::move(other));
  std::swap(data_, new_ptr.data_);
  return *this;
}

template <typename T>
SharedPtr<T> WeakPtr<T>::lock() const {
  ++data_->shared_ptr_number;
  return SharedPtr<T>(true, static_cast<T*>(data_->get_object()), data_);
}

template <typename T, typename Allocator, typename... Args>
SharedPtr<T> allocateShared(const Allocator& allocator, Args&&... args) {
  using ControlBlockAllocator = typename std::allocator_traits<Allocator>::
                                template rebind_alloc<FunctionalControlBlock<T, Allocator>>;
  using ControlBlockTraits = std::allocator_traits<ControlBlockAllocator>;
  auto concrete_block_allocator = static_cast<ControlBlockAllocator>(allocator);
  auto new_data = ControlBlockTraits::allocate(concrete_block_allocator, 1);
  ControlBlockTraits::construct(concrete_block_allocator, new_data, allocator,
                                std::forward<Args>(args)...);
  ++new_data->shared_ptr_number;
  return SharedPtr<T>(true, &(new_data->object), new_data);
}

template <typename T, typename... Args>
SharedPtr<T> makeShared(Args&&... args) {
  auto new_data = new FunctionalControlBlock<T, std::allocator<T>>(std::allocator<T>(),
                                                                   std::forward<Args>(args)...);
  ++new_data->shared_ptr_number;
  auto shared_ptr = SharedPtr<T>(true, &(new_data->object), new_data);
  if constexpr (std::is_base_of_v<EnableSharedFromThis<T>, T>) {
    shared_ptr->weak_ptr_ = shared_ptr;
  }
  return shared_ptr;
}

template <typename T>
class EnableSharedFromThis {
private:
  WeakPtr<T> weak_ptr_;

  template <typename Y>
  friend class SharedPtr;
  template <typename Y, typename... Args>
  friend SharedPtr<Y> makeShared(Args&&... args);

protected:
  EnableSharedFromThis() = default;

public:
  SharedPtr<T> shared_from_this() {
    if (weak_ptr_.data_->get_object() == nullptr) {
      throw std::bad_weak_ptr();
    }
    return weak_ptr_.lock();
  }
};