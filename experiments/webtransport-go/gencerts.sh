#!/usr/bin/env bash
# Generate self-signed certificates for use in localhost WebTransport server.
# This is only needed for Chrome-based browsers right now because they follow
# the specification strictly and don't seem to respect my local mkcert CA.

#######################################################################################
#                                                                                     #
# From https://w3c.github.io/webtransport/#custom-certificate-requirements:           #
#                                                                                     #
#   The custom certificate requirements are as follows: the certificate MUST be an    #
#   X.509v3 certificate as defined in [RFC5280], the key used in the Subject Public   #
#   Key field MUST be one of the allowed public key algorithms, the current time      #
#   MUST be within the validity period of the certificate as defined in Section       #
#   4.1.2.5 of [RFC5280] and the total length of the validity period MUST NOT         #
#   exceed two weeks. The user agent MAY impose additional implementation-defined     #
#   requirements on the certificate.                                                  #
#                                                                                     #
#   The exact list of allowed public key algorithms used in the Subject Public Key    #
#   Info field (and, as a consequence, in the TLS CertificateVerify message) is       #
#   implementation-defined; however, it MUST include ECDSA with the secp256r1         #
#   (NIST P-256) named group ([RFC3279], Section 2.3.5; [RFC8422]) to provide an      #
#   interoperable default. It MUST NOT contain RSA keys ([RFC3279], Section 2.3.1).   #
#                                                                                     #
#######################################################################################

openssl ecparam -name prime256v1 -genkey -out localhost.key
openssl req -new -sha256 -x509 -key localhost.key -out localhost.crt -days 10 \
  -subj "/OU=Localhost WebTransport Server/CN=localhost" \
  -addext "subjectAltName = DNS:localhost, IP:127.0.0.1, IP:0:0:0:0:0:0:0:1" \
  -addext "certificatePolicies = 1.2.3.4"
openssl x509 -text -noout -in localhost.crt
echo
openssl x509 -in localhost.crt -outform der | openssl dgst -sha256