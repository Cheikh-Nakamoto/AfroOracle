// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title AgriculturalData
/// @notice On-chain settlement store for local agricultural prices and rainfall.
/// Stores only settlement data: value (x100), timestamp, confidence, source,
/// consensus hash and the AI report hash (tamper-evidence link to off-chain proof).
contract AgriculturalData {
    struct Record {
        uint256 valueX100;
        uint256 timestamp;
        uint256 confidence;
        string source;
        string consensusHash;
        string aiReportHash;
    }

    address public publisher;

    mapping(bytes32 => Record) private records;
    mapping(bytes32 => uint256) private rainfall;

    event Published(
        string zone,
        string produit,
        string date,
        uint256 valueX100,
        uint256 confidence,
        string source,
        string consensusHash,
        string aiReportHash
    );

    constructor() {
        publisher = msg.sender;
    }

    modifier onlyPublisher() {
        require(msg.sender == publisher, "only publisher");
        _;
    }

    function publish(
        string memory zone,
        string memory produit,
        string memory date,
        uint256 valueX100,
        uint256 confidence,
        string memory source,
        string memory consensusHash,
        string memory aiReportHash
    ) external onlyPublisher {
        bytes32 key = keccak256(abi.encodePacked(zone, produit, date));
        records[key] = Record(
            valueX100,
            block.timestamp,
            confidence,
            source,
            consensusHash,
            aiReportHash
        );
        emit Published(
            zone,
            produit,
            date,
            valueX100,
            confidence,
            source,
            consensusHash,
            aiReportHash
        );
    }

    function getPrice(
        string memory zone,
        string memory produit,
        string memory date
    ) external view returns (uint256 valueX100, uint256 timestamp, uint256 confidence) {
        bytes32 key = keccak256(abi.encodePacked(zone, produit, date));
        Record storage r = records[key];
        return (r.valueX100, r.timestamp, r.confidence);
    }

    function setRainfall(string memory zone, string memory date, uint256 valueX100)
        external
        onlyPublisher
    {
        bytes32 key = keccak256(abi.encodePacked(zone, date));
        rainfall[key] = valueX100;
    }

    function getRainfall(string memory zone, string memory date)
        external
        view
        returns (uint256)
    {
        bytes32 key = keccak256(abi.encodePacked(zone, date));
        return rainfall[key];
    }
}
