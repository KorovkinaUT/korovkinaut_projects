from random import choice


def game(cave, player):
    while True:
        output_state(cave, player)
        action = input_action()
        while action not in ["shoot", "move", "quit", "restart"]:
            if action == "help":
                with open('Help.txt', 'r') as file:
                    file = file.read()
                print(file)
            elif action == "rules":
                with open('Rules.txt', 'r') as file:
                    file = file.read()
                print(file)
            print(">>>", end=" ")
            action = input_action()

        if action in ["quit", "restart"]:
            return action
        if action == "move":
            player.move(move_input(cave, player))
        elif action == "shoot":
            arrow_location = player.shoot(cave, shoot_input())
            if arrow_location == player.location:
                print("You are killed by arrow.")
                return "lose"
            if arrow_location in cave.wumpuses:
                cave.wumpuses.remove(arrow_location)
                print("You killed one Wumpus!")
            cave.move_wumpuses()

        if len(cave.wumpuses) == 0:
            print("You killed all Wumpuses. Congratulations!")
            return "win"
        if player.arrows == 0:
            print("Arrows run out.")
            return "lose"
        while player.location in cave.bats or player.location in cave.holes or player in cave.wumpuses:
            if player.location in cave.wumpuses:
                print("You are killed by the Wumpus.")
                return "lose"
            if player.location in cave.holes:
                print("You fell in hole.")
                return "lose"
            print("Bats carry you.")
            player.location = choice(range(1, 21))


def output_state(cave, player):
    print(f"You are in {player.location} room.")
    arrow_location = 0
    adjacent_rooms = cave.adjacent_rooms(player.location)
    print("Tunnels lead to", end='')
    for room in adjacent_rooms:
        print(f" {room}", end='')
    print(".")
    print(f"You have {player.arrows} arrows.")

    if len(adjacent_rooms & cave.wumpuses) > 0:
        print("You can feel the stench.")
    if len(adjacent_rooms & cave.bats) > 0:
        print("You can hear some noise.")
    if len(adjacent_rooms & cave.holes) > 0:
        print("You can feel the wind.")


def input_action():
    print("Shoot or move?\n>>>", end=" ")
    action = input()
    while action not in ["shoot", "move", "quit", "rules", "help", "restart"]:
        if action == "configure":
            print("You can't configure the game while you are playing.\n>>>", end=" ")
        else:
            print("Enter correct action. To find out the list of actions enter \"help\".\n>>>", end=" ")
        action = input()
    return action


def move_input(cave, player):
    print("Where?\n>>>", end=" ")
    destination = input()
    while destination not in [str(room) for room in cave.adjacent_rooms(player.location)]:
        print("Enter the correct room number.\n>>>", end=" ")
        destination = input()
    return int(destination)


def is_way_correct(way):
    way = way.split()
    for room in way:
        if not room.isdigit():
            return False
    return True


def shoot_input():
    print("Enter a sequence of rooms separated by a space.\n>>>", end=" ")
    way = input()
    while not is_way_correct(way):
        print("Enter a sequence of rooms separated by a space.\n>>>", end=" ")
        way = input()
    return list(map(int, way.split()))
