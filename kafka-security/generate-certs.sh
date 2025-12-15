#!/bin/bash

# Generate self-signed certificates for Kafka SSL
set -e

CERT_DIR="/home/gospodin/Рабочий стол/homeProjects/ServerEye/kafka-security"
CA_DIR="$CERT_DIR/ca"

echo "Generating Kafka SSL certificates..."

# Create directories
mkdir -p "$CA_DIR"
mkdir -p "$CERT_DIR/server"

# Generate CA key and certificate
echo "Generating CA..."
openssl req -new -x509 -keyout "$CA_DIR/ca-key" -out "$CA_DIR/ca-cert" -days 365 -nodes \
    -subj "/CN=Kafka-CA/OU=ServerEye/O=ServerEye/L=Local/ST=State/C=RU"

# Generate server keystore and CSR
echo "Generating server keystore..."
keytool -keystore "$CERT_DIR/server/kafka.server.keystore.jks" -alias localhost -validity 365 -genkey -keyalg RSA \
    -storepass changeit -keypass changeit -dname "CN=localhost, OU=ServerEye, O=ServerEye, L=Local, ST=State, C=RU"

# Generate certificate signing request
echo "Generating CSR..."
keytool -keystore "$CERT_DIR/server/kafka.server.keystore.jks" -alias localhost -certreq -file "$CERT_DIR/server/server-cert-file" \
    -storepass changeit -keypass changeit

# Sign the CSR with CA
echo "Signing certificate with CA..."
openssl x509 -req -CA "$CA_DIR/ca-cert" -CAkey "$CA_DIR/ca-key" -in "$CERT_DIR/server/server-cert-file" \
    -out "$CERT_DIR/server/server-cert-signed" -days 365 -CAcreateserial -passin pass:changeit

# Import CA certificate into keystore
echo "Importing CA certificate..."
keytool -keystore "$CERT_DIR/server/kafka.server.keystore.jks" -alias CARoot -import -file "$CA_DIR/ca-cert" \
    -storepass changeit -keypass changeit -noprompt

# Import signed certificate into keystore
echo "Importing signed certificate..."
keytool -keystore "$CERT_DIR/server/kafka.server.keystore.jks" -alias localhost -import -file "$CERT_DIR/server/server-cert-signed" \
    -storepass changeit -keypass changeit -noprompt

# Create truststore for clients
echo "Creating client truststore..."
keytool -keystore "$CERT_DIR/server/kafka.client.truststore.jks" -alias CARoot -import -file "$CA_DIR/ca-cert" \
    -storepass changeit -keypass changeit -noprompt

# Export server certificate for clients
keytool -keystore "$CERT_DIR/server/kafka.server.keystore.jks" -alias localhost -exportcert -rfc -file "$CERT_DIR/server/server-cert.pem" \
    -storepass changeit -keypass changeit

# Convert server certificate to PEM format for external clients
openssl x509 -in "$CERT_DIR/server/server-cert.pem" -out "$CERT_DIR/server/server-cert.pem" -inform PEM -outform PEM

echo "Certificates generated successfully!"
echo "Files created:"
echo "  - Server keystore: $CERT_DIR/server/kafka.server.keystore.jks"
echo "  - Client truststore: $CERT_DIR/server/kafka.client.truststore.jks"
echo "  - CA certificate: $CA_DIR/ca-cert"
echo "  - Server certificate: $CERT_DIR/server/server-cert.pem"
