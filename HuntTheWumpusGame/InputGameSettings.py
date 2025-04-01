def input_game_settings():
    print("Enter the number of Wumpuses.\n>>>", end=" ")
    wumpuses = input()
    while not wumpuses.isdigit():
        if wumpuses == "quit":
            return [0, 0]
        print("Enter the correct number.\n>>>", end=" ")
        wumpuses = input()

    print("Enter the number of bats.\n>>>", end=" ")
    bats = input()
    while not bats.isdigit():
        if bats == "quit":
            return [0, 0]
        print("Enter the correct number.\n>>>", end=" ")
        bats = input()

    wumpuses = int(wumpuses)
    wumpuses = max(wumpuses, 1)
    wumpuses = min(wumpuses, 19)
    bats = int(bats)
    bats = max(bats, 2)
    bats = min(bats, 17)
    return [wumpuses, bats]
