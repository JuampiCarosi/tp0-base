import socket
import logging

from common.utils import Bet, store_bets


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self.running = True
        self.client_sock = None

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """

        # TODO: Modify this program to handle signal to graceful shutdown
        # the server
        while self.running:
            client_sock = self.__accept_new_connection()
            self.client_sock = client_sock
            self.__handle_client_connection()
            self.client_sock = None

    def shutdown(self):
        self.running = False
        if self.client_sock:
            self.client_sock.close()
            logging.info(f"action: connection closed | result: success | connection: {self.client_sock.getsockname()}")

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            res = self.client_sock.recv(1024).rstrip().decode('utf-8')
            while res.count(" ") < 1:
                res += self.client_sock.recv(1024).rstrip().decode('utf-8')
            split = res.split(" ", 1)
            number = int(split[0])
            msg = split[1]

            while len(msg) < number:
                msg += self.client_sock.recv(1024).rstrip().decode('utf-8')
            addr = self.client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]}')

            bet = parse_bet(msg)
            try:
                store_bets([bet])
                logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")
                self.__send_response_safe(b"OK\n")
            except Exception as e:
                logging.error(f"action: apuesta_almacenada | result: fail | error: {e}")
                self.__send_response_safe(b"ERROR SAVING BET\n")

        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            self.client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def __send_response_safe(self, response):
        sent = 0
        while sent < len(response):
            sent += self.client_sock.send(response[sent:])


def parse_bet(msg):
    agency = msg.split(";")[0]
    name = msg.split(";")[1]
    sur_name = msg.split(";")[2]
    document = msg.split(";")[3]
    birthdate = msg.split(";")[4].split(" ")[0]
    number = msg.split(";")[5]
    return Bet(agency, name, sur_name, document, birthdate, number)
