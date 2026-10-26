// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IAgri {
    function getPrice(string memory zone, string memory produit, string memory date)
        external
        view
        returns (uint256 valueX100, uint256 timestamp, uint256 confidence);
}

/// @title ParametricInsurance
/// @notice Demo parametric cover that reads a published price from AgriculturalData.
/// Pays out when the price falls below the insured floor (thresholdX100).
contract ParametricInsurance {
    IAgri public oracle;
    address public insurer;
    uint256 public thresholdX100;

    constructor(address _oracle, uint256 _thresholdX100) {
        oracle = IAgri(_oracle);
        insurer = msg.sender;
        thresholdX100 = _thresholdX100;
    }

    /// @param zone  e.g. "Kaolack"
    /// @param date  e.g. "2026-10-26"
    /// @return payout  parametric payout in x100 units (0 when price >= threshold)
    function check(string memory zone, string memory date) external view returns (uint256 payout) {
        (uint256 price, , ) = oracle.getPrice(zone, "millet", date);
        if (price >= thresholdX100) {
            return 0; // price at or above the insured floor: no indemnification
        }
        return thresholdX100 - price;
    }
}
